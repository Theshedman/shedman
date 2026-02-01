package snapshot

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/executor"
)

const (
	zfsPropertyDescription = "com.shedman:description"
	zfsPropertyType        = "com.shedman:type"
	zfsPropertyID          = "com.shedman:id"
)

// ZFSBackend implements SnapshotManager using the 'zfs' CLI
type ZFSBackend struct {
	cfg         *config.Config
	exec        executor.Executor
	rootDataset string
}

type snapshotRef struct {
	Name    string
	Dataset string
	ID      string
}

type zfsStreamInfo struct {
	Dataset string
	ID      string
}

// NewZFSBackend creates a new ZFS backend
func NewZFSBackend(cfg *config.Config, exec executor.Executor) *ZFSBackend {
	if exec == nil {
		exec = &executor.RealExecutor{}
	}
	return &ZFSBackend{
		cfg:  cfg,
		exec: exec,
	}
}

func (b *ZFSBackend) GetBackendName() string {
	return "zfs"
}

// SetRootDataset sets a fixed root dataset (for testing).
func (b *ZFSBackend) SetRootDataset(name string) {
	b.rootDataset = name
}

// Create creates a new snapshot using zfs snapshot.
func (b *ZFSBackend) Create(ctx context.Context, desc string, opts CreateOptions) (*Snapshot, error) {
	id := time.Now().Format("20060102-150405")

	datasets, root, err := b.resolveDatasets(opts)
	if err != nil {
		return nil, err
	}

	if opts.DryRun {
		output.Info("Dry-run: zfs snapshot for datasets %v", datasets)
		return &Snapshot{
			ID:          "dry-run",
			Description: desc,
			Timestamp:   time.Now(),
			Backend:     "zfs",
			Type:        opts.Type,
		}, nil
	}

	recursive := len(opts.TargetConfigs) == 0
	for _, dataset := range datasets {
		args := []string{"snapshot"}
		if recursive && (dataset == root || strings.HasPrefix(dataset, root+"/")) {
			args = append(args, "-r")
		} else if recursive && root != "" && !strings.HasPrefix(dataset, root+"/") {
			args = append(args, "-r")
		}
		if desc != "" {
			args = append(args, "-o", fmt.Sprintf("%s=%s", zfsPropertyDescription, desc))
		}
		if opts.Type != "" {
			args = append(args, "-o", fmt.Sprintf("%s=%s", zfsPropertyType, opts.Type))
		}
		args = append(args, "-o", fmt.Sprintf("%s=%s", zfsPropertyID, id))
		args = append(args, fmt.Sprintf("%s@%s", dataset, id))

		cmd := b.zfsCommand(ctx, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("zfs snapshot failed for %s: %w", dataset, err)
		}
	}

	return &Snapshot{
		ID:          id,
		Description: desc,
		Timestamp:   time.Now(),
		Backend:     "zfs",
		Type:        opts.Type,
	}, nil
}

// List lists local or remote snapshots.
func (b *ZFSBackend) List(ctx context.Context, opts ListOptions) ([]Snapshot, error) {
	if opts.Target != nil {
		return b.listRemote(ctx, opts.Target)
	}

	records, err := b.listSnapshotRecords(ctx)
	if err != nil {
		return nil, err
	}

	root, _ := b.rootDatasetName()
	return parseZfsSnapshots(records, root), nil
}

// Delete deletes a snapshot by ID.
func (b *ZFSBackend) Delete(ctx context.Context, id string) error {
	refs, err := b.findSnapshotsByID(ctx, id)
	if err != nil {
		return err
	}

	targets := topLevelSnapshotRefs(refs)
	for _, ref := range targets {
		cmd := b.zfsCommand(ctx, "destroy", "-r", ref.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("zfs destroy failed for %s: %w", ref.Name, err)
		}
	}
	return nil
}

// Restore rolls back to a snapshot.
func (b *ZFSBackend) Restore(ctx context.Context, id string, opts RestoreOptions) error {
	if opts.DryRun {
		output.Info("Dry-run: zfs rollback -r %s", id)
		return nil
	}

	refs, err := b.findSnapshotsByID(ctx, id)
	if err != nil {
		return err
	}

	targets := topLevelSnapshotRefs(refs)
	for _, ref := range targets {
		cmd := b.zfsCommand(ctx, "rollback", "-r", ref.Name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("zfs rollback failed for %s: %w", ref.Name, err)
		}
	}
	return nil
}

// Prune removes older snapshots based on count.
func (b *ZFSBackend) Prune(ctx context.Context, opts PruneOptions) error {
	snaps, err := b.List(ctx, ListOptions{})
	if err != nil {
		return err
	}

	if opts.KeepLast > 0 && len(snaps) > opts.KeepLast {
		toDelete := snaps[opts.KeepLast:]
		for _, s := range toDelete {
			if err := b.Delete(ctx, s.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// Push sends a snapshot to a remote or local target.
func (b *ZFSBackend) Push(ctx context.Context, id string, target RemoteTarget, opts RemoteOptions) error {
	if opts.DryRun {
		output.Info("Dry-run: zfs send for snapshot %s", id)
		return nil
	}

	refs, err := b.findSnapshotsByID(ctx, id)
	if err != nil {
		return err
	}

	targets := topLevelSnapshotRefs(refs)
	localRoot, _ := b.rootDatasetName()

	switch strings.ToLower(target.Type) {
	case "ssh":
		host, dataset, err := parseSSHPath(target.Path)
		if err != nil {
			return err
		}
		for _, ref := range targets {
			remoteDataset := resolveRemoteDataset(ref.Dataset, localRoot, dataset)
			if err := b.sendOverSSH(ctx, ref.Name, host, remoteDataset); err != nil {
				return err
			}
		}
		return nil
	case StrategyLocal:
		for _, ref := range targets {
			if err := b.sendToLocalPath(ctx, ref.Name, ref.Dataset, target.Path, id); err != nil {
				return err
			}
		}
		return nil
	default:
		for _, ref := range targets {
			if err := b.sendViaRclone(ctx, ref.Name, ref.Dataset, target.Path, id, opts); err != nil {
				return err
			}
		}
		return nil
	}
}

// Pull receives a snapshot from a remote or local target.
func (b *ZFSBackend) Pull(ctx context.Context, id string, source RemoteTarget, opts RemoteOptions) error {
	if opts.DryRun {
		output.Info("Dry-run: zfs receive for snapshot %s", id)
		return nil
	}

	root, err := b.rootDatasetName()
	if err != nil {
		return err
	}

	switch strings.ToLower(source.Type) {
	case "ssh":
		host, remoteRoot, err := parseSSHPath(source.Path)
		if err != nil {
			return err
		}
		remoteRefs, err := b.findRemoteSnapshotsByID(ctx, host, id)
		if err != nil {
			return err
		}
		targets := topLevelSnapshotRefs(remoteRefs)
		for _, ref := range targets {
			destDataset := mapRemoteToLocal(ref.Dataset, root, remoteRoot)
			if err := b.receiveOverSSH(ctx, host, ref.Dataset, id, destDataset); err != nil {
				return err
			}
		}
		return nil
	case StrategyLocal:
		streams, err := b.listLocalStreams(source.Path, id)
		if err != nil {
			return err
		}
		if len(streams) == 0 {
			return fmt.Errorf("snapshot %s not found in local path", id)
		}
		for _, stream := range streams {
			destDataset := stream.Dataset
			if destDataset == "" {
				destDataset = root
			}
			if err := b.receiveFromLocalPath(ctx, source.Path, stream.Dataset, id, destDataset); err != nil {
				return err
			}
		}
		return nil
	default:
		streams, err := b.listRemoteStreams(ctx, source.Path, id)
		if err != nil {
			return err
		}
		if len(streams) == 0 {
			return fmt.Errorf("snapshot %s not found in remote path", id)
		}
		for _, stream := range streams {
			destDataset := stream.Dataset
			if destDataset == "" {
				destDataset = root
			}
			if err := b.receiveViaRclone(ctx, source.Path, stream.Dataset, id, destDataset, opts); err != nil {
				return err
			}
		}
		return nil
	}
}

// Diff compares two snapshots using zfs diff.
func (b *ZFSBackend) Diff(ctx context.Context, id1, id2 string) (DiffResult, error) {
	left, err := b.snapshotMapByDataset(ctx, id1)
	if err != nil {
		return DiffResult{}, err
	}
	right, err := b.snapshotMapByDataset(ctx, id2)
	if err != nil {
		return DiffResult{}, err
	}

	var missing []string
	for dataset := range left {
		if _, ok := right[dataset]; !ok {
			missing = append(missing, dataset)
		}
	}
	for dataset := range right {
		if _, ok := left[dataset]; !ok {
			missing = append(missing, dataset)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return DiffResult{}, fmt.Errorf("snapshot sets do not match for datasets: %s", strings.Join(missing, ", "))
	}

	res := DiffResult{}
	for dataset, snap1 := range left {
		snap2 := right[dataset]
		out, err := b.zfsOutput("diff", "-H", snap1, snap2)
		if err != nil {
			return DiffResult{}, fmt.Errorf("zfs diff failed for %s: %w", dataset, err)
		}

		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			change := parts[0]
			path := strings.Join(parts[1:], " ")
			switch change {
			case "+":
				res.Added = append(res.Added, path)
			case "-":
				res.Removed = append(res.Removed, path)
			case "M":
				res.Modified = append(res.Modified, path)
			}
		}
	}

	return res, nil
}

func (b *ZFSBackend) listRemote(ctx context.Context, target *RemoteTarget) ([]Snapshot, error) {
	if target == nil || target.Path == "" {
		return nil, fmt.Errorf("remote target path is required")
	}

	args := []string{"lsjson", "--files-only", target.Path}
	cmdArgs := util.GetPrivilegedRcloneCommand(args)
	out, err := b.exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...).Output()
	if err != nil {
		return nil, fmt.Errorf("rclone listing failed: %w", err)
	}

	var entries []struct {
		Name    string    `json:"Name"`
		ModTime time.Time `json:"ModTime"`
		IsDir   bool      `json:"IsDir"`
	}
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse rclone output: %w", err)
	}

	var results []Snapshot
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		dataset, id, ok := parseStreamFilename(entry.Name)
		if !ok {
			continue
		}
		ts := entry.ModTime
		if parsed, err := time.Parse("20060102-150405", id); err == nil {
			ts = parsed
		}
		results = append(results, Snapshot{
			ID:          id,
			Timestamp:   ts,
			Backend:     "zfs",
			Description: "Remote ZFS Snapshot",
			Tags:        []string{dataset},
		})
	}

	return results, nil
}

func (b *ZFSBackend) resolveDatasets(opts CreateOptions) ([]string, string, error) {
	if len(opts.TargetConfigs) > 0 {
		return opts.TargetConfigs, opts.TargetConfigs[0], nil
	}

	root, err := b.rootDatasetName()
	if err != nil {
		return nil, "", err
	}
	if root == "" {
		return nil, "", fmt.Errorf("root dataset not found")
	}

	datasets := []string{root}
	if opts.IncludeHome {
		home, _ := b.datasetByMountpoint("/home")
		if home != "" && home != root && !strings.HasPrefix(home, root+"/") {
			datasets = append(datasets, home)
		}
	}

	return datasets, root, nil
}

func (b *ZFSBackend) listSnapshotRecords(ctx context.Context) (string, error) {
	_ = ctx
	columns := fmt.Sprintf("name,creation,used,%s,%s,%s", zfsPropertyDescription, zfsPropertyType, zfsPropertyID)
	out, err := b.zfsOutput("list", "-H", "-p", "-t", "snapshot", "-o", columns)
	if err != nil {
		return "", fmt.Errorf("zfs list failed: %w", err)
	}
	return string(out), nil
}

func (b *ZFSBackend) listSnapshotRefs(ctx context.Context) ([]snapshotRef, error) {
	_ = ctx
	columns := fmt.Sprintf("name,%s", zfsPropertyID)
	out, err := b.zfsOutput("list", "-H", "-t", "snapshot", "-o", columns)
	if err != nil {
		return nil, fmt.Errorf("zfs list failed: %w", err)
	}
	return parseSnapshotRefs(string(out)), nil
}

func parseSnapshotRefs(output string) []snapshotRef {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	refs := make([]snapshotRef, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			parts = strings.Fields(line)
		}
		if len(parts) == 0 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		propID := ""
		if len(parts) > 1 {
			propID = strings.TrimSpace(parts[1])
			if propID == "-" {
				propID = ""
			}
		}
		dataset := name
		if at := strings.Index(name, "@"); at != -1 {
			dataset = name[:at]
		}
		refs = append(refs, snapshotRef{
			Name:    name,
			Dataset: dataset,
			ID:      propID,
		})
	}
	return refs
}

func (b *ZFSBackend) findSnapshotsByID(ctx context.Context, id string) ([]snapshotRef, error) {
	refs, err := b.listSnapshotRefs(ctx)
	if err != nil {
		return nil, err
	}
	nameMatches, propMatches := filterSnapshotRefsByID(refs, id)
	if len(propMatches) == 0 && len(nameMatches) == 0 {
		return nil, fmt.Errorf("snapshot %s not found", id)
	}
	if len(propMatches) == 0 {
		return nameMatches, nil
	}
	return mergeSnapshotRefs(propMatches, nameMatches), nil
}

func (b *ZFSBackend) snapshotMapByDataset(ctx context.Context, id string) (map[string]string, error) {
	refs, err := b.findSnapshotsByID(ctx, id)
	if err != nil {
		return nil, err
	}
	matches := make(map[string]string)
	for _, ref := range refs {
		if ref.Dataset == "" {
			continue
		}
		if _, ok := matches[ref.Dataset]; !ok {
			matches[ref.Dataset] = ref.Name
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("snapshot %s not found", id)
	}
	return matches, nil
}

func filterSnapshotRefsByID(refs []snapshotRef, id string) ([]snapshotRef, []snapshotRef) {
	var nameMatches []snapshotRef
	var propMatches []snapshotRef
	for _, ref := range refs {
		if ref.ID == id {
			propMatches = append(propMatches, ref)
			continue
		}
		if ref.ID == "" && snapshotIDFromName(ref.Name) == id {
			nameMatches = append(nameMatches, ref)
		}
	}
	return nameMatches, propMatches
}

func mergeSnapshotRefs(primary, secondary []snapshotRef) []snapshotRef {
	seen := make(map[string]bool, len(primary))
	merged := make([]snapshotRef, 0, len(primary)+len(secondary))
	for _, ref := range primary {
		merged = append(merged, ref)
		seen[ref.Dataset] = true
	}
	for _, ref := range secondary {
		if !seen[ref.Dataset] {
			merged = append(merged, ref)
			seen[ref.Dataset] = true
		}
	}
	return merged
}

func topLevelSnapshotRefs(refs []snapshotRef) []snapshotRef {
	if len(refs) == 0 {
		return nil
	}
	unique := make(map[string]snapshotRef, len(refs))
	for _, ref := range refs {
		if ref.Dataset == "" {
			continue
		}
		if _, ok := unique[ref.Dataset]; !ok {
			unique[ref.Dataset] = ref
		}
	}

	candidates := make([]snapshotRef, 0, len(unique))
	for _, ref := range unique {
		candidates = append(candidates, ref)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if len(candidates[i].Dataset) == len(candidates[j].Dataset) {
			return candidates[i].Dataset < candidates[j].Dataset
		}
		return len(candidates[i].Dataset) < len(candidates[j].Dataset)
	})

	var top []snapshotRef
	for _, ref := range candidates {
		if hasParentDataset(top, ref.Dataset) {
			continue
		}
		top = append(top, ref)
	}
	return top
}

func hasParentDataset(refs []snapshotRef, dataset string) bool {
	for _, ref := range refs {
		if ref.Dataset == dataset || strings.HasPrefix(dataset, ref.Dataset+"/") {
			return true
		}
	}
	return false
}

func resolveRemoteDataset(localDataset, localRoot, remoteRoot string) string {
	remoteRoot = strings.TrimSuffix(remoteRoot, "/")
	localRoot = strings.TrimSuffix(localRoot, "/")

	if remoteRoot == "" {
		return localDataset
	}
	if localRoot == "" {
		return remoteRoot + "/" + localDataset
	}
	if localDataset == localRoot {
		return remoteRoot
	}
	if strings.HasPrefix(localDataset, localRoot+"/") {
		return remoteRoot + "/" + strings.TrimPrefix(localDataset, localRoot+"/")
	}
	return remoteRoot + "/" + localDataset
}

func mapRemoteToLocal(remoteDataset, localRoot, remoteRoot string) string {
	remoteRoot = strings.TrimSuffix(remoteRoot, "/")
	localRoot = strings.TrimSuffix(localRoot, "/")

	if remoteRoot == "" {
		return remoteDataset
	}
	if remoteDataset == remoteRoot {
		if localRoot != "" {
			return localRoot
		}
		return remoteDataset
	}
	if strings.HasPrefix(remoteDataset, remoteRoot+"/") {
		suffix := strings.TrimPrefix(remoteDataset, remoteRoot+"/")
		if localRoot == "" {
			return suffix
		}
		localPool := strings.SplitN(localRoot, "/", 2)[0]
		if localPool != "" && (suffix == localPool || strings.HasPrefix(suffix, localPool+"/")) {
			return suffix
		}
		return localRoot + "/" + suffix
	}
	return remoteDataset
}

func (b *ZFSBackend) findRemoteSnapshotsByID(ctx context.Context, host, id string) ([]snapshotRef, error) {
	if host == "" {
		return nil, fmt.Errorf("ssh host is required")
	}
	columns := fmt.Sprintf("name,%s", zfsPropertyID)
	args := []string{"sudo", "-n", "zfs", "list", "-H", "-t", "snapshot", "-o", columns}
	cmd := b.exec.CommandContext(ctx, "ssh", append([]string{host}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("remote zfs list failed: %w", err)
	}
	refs := parseSnapshotRefs(string(out))
	nameMatches, propMatches := filterSnapshotRefsByID(refs, id)
	if len(propMatches) == 0 && len(nameMatches) == 0 {
		return nil, fmt.Errorf("snapshot %s not found", id)
	}
	if len(propMatches) == 0 {
		return nameMatches, nil
	}
	return mergeSnapshotRefs(propMatches, nameMatches), nil
}

func (b *ZFSBackend) listLocalStreams(path, id string) ([]zfsStreamInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var streams []zfsStreamInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		dataset, streamID, ok := parseStreamFilename(entry.Name())
		if !ok {
			continue
		}
		if id != "" && streamID != id {
			continue
		}
		streams = append(streams, zfsStreamInfo{
			Dataset: dataset,
			ID:      streamID,
		})
	}
	return streams, nil
}

func (b *ZFSBackend) listRemoteStreams(ctx context.Context, path, id string) ([]zfsStreamInfo, error) {
	if path == "" {
		return nil, fmt.Errorf("remote target path is required")
	}
	args := []string{"lsjson", "--files-only", path}
	cmdArgs := util.GetPrivilegedRcloneCommand(args)
	out, err := b.exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...).Output()
	if err != nil {
		return nil, fmt.Errorf("rclone listing failed: %w", err)
	}

	var entries []struct {
		Name  string `json:"Name"`
		IsDir bool   `json:"IsDir"`
	}
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse rclone output: %w", err)
	}

	var streams []zfsStreamInfo
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		dataset, streamID, ok := parseStreamFilename(entry.Name)
		if !ok {
			continue
		}
		if id != "" && streamID != id {
			continue
		}
		streams = append(streams, zfsStreamInfo{
			Dataset: dataset,
			ID:      streamID,
		})
	}
	return streams, nil
}

func (b *ZFSBackend) rootDatasetName() (string, error) {
	if b.rootDataset != "" {
		return b.rootDataset, nil
	}
	return b.datasetByMountpoint("/")
}

func (b *ZFSBackend) datasetByMountpoint(mountpoint string) (string, error) {
	out, err := b.zfsOutput("list", "-H", "-o", "name,mountpoint")
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			parts = strings.Fields(line)
		}
		if len(parts) < 2 {
			continue
		}
		if strings.TrimSpace(parts[1]) == mountpoint {
			return strings.TrimSpace(parts[0]), nil
		}
	}

	return "", fmt.Errorf("dataset mounted at %s not found", mountpoint)
}

func (b *ZFSBackend) resolveSnapshotName(id string) (string, error) {
	if strings.Contains(id, "@") {
		return id, nil
	}
	root, err := b.rootDatasetName()
	if err != nil {
		return "", err
	}
	if root == "" {
		return "", fmt.Errorf("root dataset not found")
	}
	return formatSnapshotName(root, id), nil
}

func (b *ZFSBackend) zfsCommand(ctx context.Context, args ...string) *exec.Cmd {
	if b.exec == nil {
		b.exec = &executor.RealExecutor{}
	}
	if os.Geteuid() != 0 {
		cmdArgs := append([]string{"-n", "zfs"}, args...)
		return b.exec.CommandContext(ctx, "sudo", cmdArgs...)
	}
	return b.exec.CommandContext(ctx, "zfs", args...)
}

func (b *ZFSBackend) zfsOutput(args ...string) ([]byte, error) {
	if b.exec == nil {
		b.exec = &executor.RealExecutor{}
	}
	if os.Geteuid() != 0 {
		cmdArgs := append([]string{"-n", "zfs"}, args...)
		return b.exec.Output("sudo", cmdArgs...)
	}
	return b.exec.Output("zfs", args...)
}

func parseZfsSnapshots(output, root string) []Snapshot {
	agg := make(map[string]*Snapshot)

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			parts = strings.Fields(line)
		}
		if len(parts) < 3 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		if root != "" && !strings.HasPrefix(name, root+"@") && !strings.HasPrefix(name, root+"/") {
			continue
		}
		snapID := snapshotIDFromName(name)

		created := parseEpoch(parts[1])
		size := parseBytes(parts[2])
		desc := getField(parts, 3)
		if desc == "-" {
			desc = ""
		}
		typ := getField(parts, 4)
		if typ == "-" {
			typ = ""
		}
		propID := getField(parts, 5)
		if propID != "" && propID != "-" {
			snapID = propID
		}

		entry, ok := agg[snapID]
		if !ok {
			entry = &Snapshot{
				ID:          snapID,
				Description: desc,
				Timestamp:   created,
				Type:        typ,
				Backend:     "zfs",
				Size:        size,
			}
			agg[snapID] = entry
		} else {
			entry.Size += size
			if entry.Description == "" {
				entry.Description = desc
			}
			if entry.Type == "" {
				entry.Type = typ
			}
			if created.After(entry.Timestamp) {
				entry.Timestamp = created
			}
		}
	}

	var snapshots []Snapshot
	for _, snap := range agg {
		snapshots = append(snapshots, *snap)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	return snapshots
}

func snapshotIDFromName(name string) string {
	parts := strings.SplitN(name, "@", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return name
}

func parseEpoch(value string) time.Time {
	ts, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

func parseBytes(value string) int64 {
	if value == "" {
		return 0
	}
	if parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
		return parsed
	}
	return util.ParseSize(value)
}

func getField(parts []string, idx int) string {
	if idx >= 0 && idx < len(parts) {
		return strings.TrimSpace(parts[idx])
	}
	return ""
}

func formatSnapshotName(root, id string) string {
	if strings.Contains(id, "@") {
		return id
	}
	return fmt.Sprintf("%s@%s", root, id)
}

func parseSSHPath(path string) (string, string, error) {
	parts := strings.SplitN(path, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid ssh target path: %s (expected host:dataset)", path)
	}
	return parts[0], parts[1], nil
}

func zfsStreamFilename(dataset, id string) string {
	escapedID := url.PathEscape(id)
	if dataset == "" {
		return fmt.Sprintf("zfs-%s.zfs", escapedID)
	}
	escapedDataset := url.PathEscape(dataset)
	return fmt.Sprintf("zfs-%s@%s.zfs", escapedDataset, escapedID)
}

func parseStreamFilename(name string) (string, string, bool) {
	if !strings.HasPrefix(name, "zfs-") || !strings.HasSuffix(name, ".zfs") {
		return "", "", false
	}
	base := strings.TrimSuffix(strings.TrimPrefix(name, "zfs-"), ".zfs")
	if base == "" {
		return "", "", false
	}
	if strings.Contains(base, "@") {
		parts := strings.SplitN(base, "@", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		dataset, err := url.PathUnescape(parts[0])
		if err != nil {
			return "", "", false
		}
		id, err := url.PathUnescape(parts[1])
		if err != nil {
			return "", "", false
		}
		return dataset, id, true
	}
	id, err := url.PathUnescape(base)
	if err != nil {
		return "", "", false
	}
	return "", id, true
}

func (b *ZFSBackend) sendToLocalPath(ctx context.Context, snapName, dataset, targetDir, id string) error {
	if err := os.MkdirAll(targetDir, util.DirPermissions); err != nil {
		return fmt.Errorf("failed to create target dir: %w", err)
	}

	filePath := filepath.Join(targetDir, zfsStreamFilename(dataset, id))
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create stream file: %w", err)
	}
	defer func() { _ = f.Close() }()

	cmd := b.zfsCommand(ctx, "send", "-R", snapName)
	cmd.Stdout = f
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("zfs send failed: %w", err)
	}
	return nil
}

func (b *ZFSBackend) sendViaRclone(ctx context.Context, snapName, dataset, targetPath, id string, opts RemoteOptions) error {
	tmpFile, err := os.CreateTemp("", "shedman-zfs-*.zfs")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	defer func() { _ = tmpFile.Close() }()

	cmd := b.zfsCommand(ctx, "send", "-R", snapName)
	cmd.Stdout = tmpFile
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("zfs send failed: %w", err)
	}

	dest := targetPath
	if !strings.HasSuffix(dest, "/") && !strings.HasSuffix(dest, ":") {
		dest += "/"
	}
	dest += zfsStreamFilename(dataset, id)

	args := []string{"copy", tmpPath, dest}
	if opts.Bandwidth > 0 {
		args = append(args, fmt.Sprintf("--bwlimit=%dK", opts.Bandwidth))
	}
	if opts.Compress {
		args = append(args, "--compress")
	}
	cmdArgs := util.GetPrivilegedRcloneCommand(args)
	out, err := b.exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone copy failed: %w\nOutput: %s", err, string(out))
	}
	return nil
}

func (b *ZFSBackend) receiveFromLocalPath(ctx context.Context, sourceDir, dataset, id, destDataset string) error {
	filePath := filepath.Join(sourceDir, zfsStreamFilename(dataset, id))
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open stream file: %w", err)
	}
	defer func() { _ = f.Close() }()

	cmd := b.zfsCommand(ctx, "receive", "-F", destDataset)
	cmd.Stdin = f
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *ZFSBackend) receiveViaRclone(ctx context.Context, sourcePath, dataset, id, destDataset string, opts RemoteOptions) error {
	tmpFile, err := os.CreateTemp("", "shedman-zfs-*.zfs")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	defer func() { _ = tmpFile.Close() }()

	src := sourcePath
	if !strings.HasSuffix(src, "/") && !strings.HasSuffix(src, ":") {
		src += "/"
	}
	src += zfsStreamFilename(dataset, id)

	args := []string{"copy", src, tmpPath}
	if opts.Bandwidth > 0 {
		args = append(args, fmt.Sprintf("--bwlimit=%dK", opts.Bandwidth))
	}
	if opts.Compress {
		args = append(args, "--compress")
	}
	cmdArgs := util.GetPrivilegedRcloneCommand(args)
	out, err := b.exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone copy failed: %w\nOutput: %s", err, string(out))
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	cmd := b.zfsCommand(ctx, "receive", "-F", destDataset)
	cmd.Stdin = tmpFile
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (b *ZFSBackend) sendOverSSH(ctx context.Context, snapName, host, dataset string) error {
	sendCmd := b.zfsCommand(ctx, "send", "-R", snapName)
	recvCmd := b.exec.CommandContext(ctx, "ssh", host, "sudo", "-n", "zfs", "receive", "-F", dataset)

	pipe, err := sendCmd.StdoutPipe()
	if err != nil {
		return err
	}
	sendCmd.Stderr = os.Stderr
	recvCmd.Stdin = pipe
	recvCmd.Stdout = os.Stdout
	recvCmd.Stderr = os.Stderr

	if err := recvCmd.Start(); err != nil {
		return err
	}
	if err := sendCmd.Start(); err != nil {
		return err
	}

	if err := sendCmd.Wait(); err != nil {
		_ = recvCmd.Wait()
		return err
	}
	return recvCmd.Wait()
}

func (b *ZFSBackend) receiveOverSSH(ctx context.Context, host, dataset, id, destDataset string) error {
	sendCmd := b.exec.CommandContext(ctx, "ssh", host, "sudo", "-n", "zfs", "send", "-R", fmt.Sprintf("%s@%s", dataset, id))
	recvCmd := b.zfsCommand(ctx, "receive", "-F", destDataset)

	pipe, err := sendCmd.StdoutPipe()
	if err != nil {
		return err
	}
	sendCmd.Stderr = os.Stderr
	recvCmd.Stdin = pipe
	recvCmd.Stdout = os.Stdout
	recvCmd.Stderr = os.Stderr

	if err := recvCmd.Start(); err != nil {
		return err
	}
	if err := sendCmd.Start(); err != nil {
		return err
	}

	if err := sendCmd.Wait(); err != nil {
		_ = recvCmd.Wait()
		return err
	}
	return recvCmd.Wait()
}

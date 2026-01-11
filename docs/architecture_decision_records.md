# Architecture Decision Records (ADRs)

## Table of Contents

1. [ADR-001: Arch-Based Linux Only](#adr-001-arch-based-linux-only)
2. [ADR-002: Modular Monorepo Architecture](#adr-002-modular-monorepo-architecture)
3. [ADR-003: Capability-Based Backend Interface](#adr-003-capability-based-backend-interface)
4. [ADR-004: 100% Pacman Compatibility](#adr-004-100-pacman-compatibility)
5. [ADR-005: No Package Version Management](#adr-005-no-package-version-management)
6. [ADR-006: go-alpm Integration with Pacman Shelling](#adr-006-go-alpm-integration-with-pacman-shelling)
7. [ADR-007: Integrate Existing Snapshot Tools](#adr-007-integrate-existing-snapshot-tools)
8. [ADR-008: Package Groups Support](#adr-008-package-groups-support)
9. [ADR-009: Simplified Command Set](#adr-009-simplified-command-set)
10. [ADR-010: No Plugin System (Initially)](#adr-010-no-plugin-system-initially)
11. [ADR-011: Production-Ready, Not MVP](#adr-011-production-ready-not-mvp)
12. [ADR-012: .shed Format for Future ShedOS Only](#adr-012-shed-format-for-future-shedos-only)

---

## ADR-001: Arch-Based Linux Only

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Original plan included cross-distribution support (Arch, Debian, Fedora, FreeBSD, Alpine) with package format conversion and universal dependency mapping.

**Challenges with cross-distro:**

- Package conversion is fundamentally lossy (debian2rpm, alien)
- Dependency mapping requires maintaining huge database (packages × distros)
- Different distros have incompatible package semantics
- Testing complexity multiplies by number of distros
- Focus divided across multiple package manager ecosystems

### Decision

**shedman will ONLY support Arch-based Linux distributions** (ShedOS, Arch Linux, Manjaro, EndeavourOS, etc.).

**Exception**: `shedman snapshot` may work on other distros for backup/restore functionality.

### Consequences

**Positive:**

- ✅ Clear target audience (Arch users)
- ✅ Single package format (.pkg.tar.zst)
- ✅ Single package manager integration (pacman/libalpm)
- ✅ Faster development (no multi-distro testing)
- ✅ Better quality (deep Arch integration)
- ✅ Simpler codebase
- ✅ Clearer documentation

**Negative:**

- ❌ Not usable by Debian/Fedora users (for package management)
- ❌ Smaller potential user base

**Mitigation:**

- Market as "Best package manager for Arch users"
- `shedman snapshot` can still provide value to non-Arch users

### Alternatives Considered

1. **Full cross-distro support**: Rejected due to complexity and lossy conversions
2. **Arch + Debian only**: Still too complex, different philosophies
3. **Universal .shed format for all distros**: Postponed to future ShedOS rebuild

---

## ADR-002: Modular Monorepo Architecture

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Need to organize a feature-rich package manager with multiple concerns:

- Core package operations
- Snapshot/backup system
- Configuration management
- Desktop environment switching
- Theme management
- Service management
- System utilities

**Options:**

1. **Single monolithic binary** - Everything in one codebase, one command
2. **Separate repositories** - Each module its own repo (shedman-core, shedman-snapshot, etc.)
3. **Modular monorepo** - Modules in single repo, compose into unified CLI

### Decision

**Use modular monorepo architecture:**

```
shedman (monorepo)
├── cmd/
│   └── shedman/          # Main CLI entry point
├── pkg/
│   ├── core/             # Package management (install, remove, update)
│   ├── snapshot/         # Backup/restore
│   ├── config/           # Config package management
│   ├── de/               # Desktop environment switching
│   ├── theme/            # Theme management
│   ├── boot/             # Boot management
│   ├── svc/              # Service management (systemd)
│   ├── notifier/         # Update notifications
│   ├── mirror/           # Mirror management
│   ├── log/              # Logging system
│   ├── tui/              # Terminal UI
│   ├── keyring/          # GPG key management
│   └── security/         # Security/CVE checking
└── internal/             # Shared utilities
    ├── alpm/             # go-alpm wrapper
    ├── config/           # Config file parsing
    └── output/           # Pretty printing
```

**Single command interface:**

```bash
shedman install pkg       # Uses pkg/core
shedman snapshot create   # Uses pkg/snapshot
shedman config apply nvim # Uses pkg/config
shedman de switch gnome   # Uses pkg/de
```

### Consequences

**Positive:**

- ✅ Clean separation of concerns
- ✅ Each module can be developed/tested independently
- ✅ Shared code in internal packages
- ✅ Single binary for users (good UX)
- ✅ Single version number
- ✅ Easier dependency management
- ✅ Atomic releases

**Negative:**

- ❌ Modules can't have independent release cycles
- ❌ One module's bug affects whole binary

**Trade-offs accepted:**

- Users get one cohesive tool
- Developers work in organized codebase

### Alternatives Considered

1. **Monolith**: Hard to maintain, tight coupling
2. **Separate repos**: Versioning hell, user installs multiple tools
3. **Plugins**: Over-engineering for core functionality

---

## ADR-003: Capability-Based Backend Interface

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Need interface for package backend (pacman/libalpm). Original design used single unified interface with all methods:

```go
type OfficialBackend interface {
    Install(...)
    Remove(...)
    Upgrade(...)
    Search(...)
    Info(...)
    // ... 10+ methods
}
```

**Problem**: Forces all backends to implement all features, even if not supported.

### Decision

**Use capability-based interface design with optional features:**

```go
// Core interface - minimal, all backends must implement
type Backend interface {
    Name() string
    Sync() error
    IsAvailable() bool
}

// Package operations - separate interface
type PackageManager interface {
    Backend
    Install(pkgs []string, opts InstallOptions) error
    Remove(pkgs []string, opts RemoveOptions) error
    IsInstalled(pkg string) bool
}

// Optional search capability
type Searchable interface {
    Search(query string) ([]PackageInfo, error)
}

// Optional information querying
type Informer interface {
    Info(pkgName string) (*PackageInfo, error)
    GetInstalledPackages() ([]PackageInfo, error)
}

// Optional upgrade capability
type Upgradable interface {
    Upgrade(pkgs []string, opts UpgradeOptions) error
}

// Optional local file installation
type LocalInstaller interface {
    InstallLocal(path string, opts InstallOptions) error
}

// Optional package file querying
type FileProvider interface {
    GetPackageFiles(pkgName string) ([]string, error)
}
```

**Usage:**

```go
backend := GetBackend()

// Always available
backend.Sync()

// Check capabilities
if pm, ok := backend.(PackageManager); ok {
    pm.Install(pkgs, opts)
}

if searcher, ok := backend.(Searchable); ok {
    results := searcher.Search(query)
} else {
    return errors.New("backend doesn't support search")
}
```

### Consequences

**Positive:**

- ✅ Backends declare actual capabilities
- ✅ No fake implementations
- ✅ Graceful feature degradation
- ✅ Easy to add new backends
- ✅ Interface segregation principle

**Negative:**

- ❌ More interfaces to manage
- ❌ Runtime type assertions needed

**Trade-offs:**

- Slightly more complex code for better flexibility

### Alternatives Considered

1. **Single unified interface**: Forces fakes, doesn't respect backend differences
2. **Backend-specific APIs**: No common abstraction
3. **Feature flags in struct**: Runtime checks, no compile-time safety

---

## ADR-004: 100% Pacman Compatibility

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Users transitioning from `pacman` expect familiar commands. Originally planned simple compatibility layer.

### Decision

**shedman must support ALL pacman flags and syntax:**

```bash
# These must work identically to pacman
shedman -Syu              # System update
shedman -S neovim         # Install
shedman -Ss vim           # Search
shedman -R firefox        # Remove
shedman -Qi neovim        # Query info (installed)
shedman -Si neovim        # Query info (sync db)
shedman -Ql neovim        # List files
shedman -Qo /usr/bin/nvim # File owner
shedman -Qdt              # List orphans
shedman -Rns pkg          # Remove with deps
```

**Implementation:**

```go
func main() {
    // Parse pacman-style flags
    if hasPacmanFlag(os.Args) {
        handlePacmanSyntax(os.Args)
    } else {
        handleShedmanSyntax(os.Args)
    }
}
```

**Config file compatibility:**

- Read `/etc/pacman.conf` directly
- Parse all pacman settings (mirrors, IgnorePkg, SigLevel, etc.)
- No migration needed

### Consequences

**Positive:**

- ✅ Zero learning curve for Arch users
- ✅ Drop-in pacman replacement
- ✅ Scripts using pacman work with shedman
- ✅ Can symlink: `ln -s shedman /usr/bin/pacman`
- ✅ Respects existing pacman configuration

**Negative:**

- ❌ Must maintain pacman flag compatibility forever
- ❌ Pacman's CLI design constraints apply to shedman

**Trade-offs:**

- Worth it for seamless migration

### Alternatives Considered

1. **Subset of pacman flags**: Confusing for users
2. **Own syntax only**: Higher barrier to adoption
3. **Alias system**: Requires user setup

---

## ADR-005: No Package Version Management

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Original plan included:

- `shedman install pkg@1.2.3` - Install specific version
- `shedman rollback pkg` - Rollback to previous version
- Version history tracking
- Package cache with multiple versions

**Reality of Arch Linux:**

- Rolling release distribution
- Only latest packages in official repos
- Users chose Arch for rolling release
- Downgrading is discouraged (may break dependencies)
- Arch Wiki: "Partial upgrades are unsupported"

### Decision

**Remove package version management features:**

```bash
# NOT supported:
shedman install neovim@0.9.0
shedman rollback neovim

# Instead:
shedman install neovim        # Always latest
shedman update                # Update all
```

**Package cache still exists:**

- pacman keeps old packages in `/var/cache/pacman/pkg/`
- Manual rollback possible: `pacman -U /var/cache/pacman/pkg/old-package.pkg.tar.zst`
- But not automated by shedman

### Consequences

**Positive:**

- ✅ Aligns with Arch philosophy
- ✅ Simpler implementation
- ✅ No version tracking database
- ✅ No version resolution conflicts
- ✅ Faster development

**Negative:**

- ❌ Can't easily rollback broken package
- ❌ Can't install old version for compatibility

**Mitigation:**

- Users can use Arch Rollback Machine (ARM) if needed
- System snapshots (`shedman snapshot`) provide full system rollback

### Alternatives Considered

1. **Full version management**: Against Arch philosophy, complex
2. **Limited history (last 3 versions)**: Still complexity for edge cases
3. **AUR-only version pinning**: Confusing partial feature

---

## ADR-006: go-alpm Integration with Pacman Shelling

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Two approaches to Arch package management:

1. **Pure go-alpm**: Use libalpm Go bindings for everything
2. **Shell to pacman**: Execute `pacman` command for operations
3. **Hybrid**: Use go-alpm for queries, pacman for installations

**Requirements:**

- Must work reliably
- Preserve pacman hooks
- Respect pacman configuration
- Handle transactions correctly

### Decision

**Use go-alpm for reads, shell to pacman for writes:**

```go
// Reading - use go-alpm (fast, direct)
func (b *Backend) Search(query string) ([]PackageInfo, error) {
    handle, err := alpm.Initialize(...)
    defer handle.Release()
    
    dbs := handle.SyncDbs()
    results := searchDatabases(dbs, query)
    return results, nil
}

func (b *Backend) IsInstalled(pkg string) bool {
    handle, err := alpm.Initialize(...)
    defer handle.Release()
    
    localDb := handle.LocalDb()
    return localDb.Pkg(pkg) != nil
}

// Writing - shell to pacman (preserve hooks, signals)
func (b *Backend) Install(pkgs []string, opts InstallOptions) error {
    args := []string{"-S"}
    if opts.NoConfirm {
        args = append(args, "--noconfirm")
    }
    args = append(args, pkgs...)
    
    cmd := exec.Command("sudo", "pacman", args...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    return cmd.Run()
}
```

### Consequences

**Positive:**

- ✅ Fast queries (go-alpm, no process spawning)
- ✅ Reliable installations (pacman handles hooks, signals)
- ✅ Preserves pacman behavior exactly
- ✅ User sees pacman progress output
- ✅ Respects all pacman configuration

**Negative:**

- ❌ Requires both go-alpm and pacman binary
- ❌ Hybrid approach (not pure library usage)

**Trade-offs:**

- Pragmatic: Use best tool for each job
- go-alpm for query performance
- pacman for installation reliability

### Alternatives Considered

1. **Pure go-alpm**: Risk missing hooks, signal handling issues
2. **Pure pacman shelling**: Slow for queries (spawn process each time)
3. **Reimplement pacman**: Years of work, likely bugs

---

## ADR-007: Integrate Existing Snapshot Tools

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

System snapshots need to:

- Support multiple filesystems (btrfs, zfs, LVM, ext4)
- Handle cloud backup (Google Drive, R2, S3)
- Provide encryption
- Be reliable

**Existing mature tools:**

- **Timeshift**: btrfs/rsync snapshots
- **snapper**: btrfs snapshots
- **zfs**: Native ZFS snapshots
- **restic**: Encrypted cloud backups
- **rclone**: Cloud storage sync

**Options:**

1. Reimplement everything in Go
2. Integrate existing tools
3. Hybrid: Own format + existing cloud tools

### Decision

**Integrate existing tools, don't reimplement:**

```bash
# shedman snapshot detects filesystem and uses appropriate tool
shedman snapshot create
  → btrfs: btrfs subvolume snapshot
  → zfs: zfs snapshot
  → LVM: lvcreate --snapshot
  → ext4: rsync + hardlinks (or call Timeshift)

# Cloud backup uses rclone
shedman snapshot push 5 --remote gdrive
  → rclone copy /snapshots/5 gdrive:shedman-backups
```

**Architecture:**

```go
type SnapshotBackend interface {
    Create(name string) error
    List() ([]Snapshot, error)
    Restore(id string) error
    Delete(id string) error
}

type BtrfsBackend struct { ... }  // calls btrfs commands
type ZfsBackend struct { ... }     // calls zfs commands
type RsyncBackend struct { ... }   // uses rsync

func DetectFilesystem() SnapshotBackend {
    // Detect and return appropriate backend
}
```

### Consequences

**Positive:**

- ✅ Leverage 20+ years of tool maturity
- ✅ Battle-tested reliability
- ✅ Less code to maintain
- ✅ Users already trust these tools
- ✅ Faster implementation
- ✅ Works on non-Arch distros (rclone, restic available everywhere)

**Negative:**

- ❌ Dependencies on external tools
- ❌ Inconsistent CLI across filesystems
- ❌ Error handling requires parsing tool output

**Mitigation:**

- Check for required tools at runtime
- Provide clear error messages if tools missing
- Document installation of dependencies

### Alternatives Considered

1. **Pure Go implementation**: Massive undertaking, likely bugs
2. **C bindings to libraries**: Complex, platform-specific
3. **Only support btrfs**: Excludes many users

---

## ADR-008: Package Groups Support

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Users want to install related packages together (e.g., development tools, gaming software, desktop environments).

**Pacman has groups:**

```bash
pacman -S base-devel  # Install development tools group
```

**User wants curated groups:**

```bash
shedman install @dev        # Development environment
shedman install @gaming     # Gaming tools
shedman install @shedos-hyprland  # ShedOS Hyprland DE
```

### Decision

**Implement package groups with @ syntax:**

```yaml
# /etc/shedman/groups/dev.yml
name: dev
description: "Development tools and compilers"
packages:
  - git
  - neovim
  - gcc
  - make
  - cmake
  - gdb
  - docker
  - base-devel

# /etc/shedman/groups/gaming.yml
name: gaming
description: "Gaming essentials"
packages:
  - steam
  - wine
  - gamemode
  - mangohud
  - lutris
```

**Commands:**

```bash
shedman install @dev                 # Install all dev packages
shedman group list                   # List available groups
shedman group info @gaming           # Show packages in gaming group
shedman group remove @gaming         # Remove all packages from group
```

**Implementation:**

```go
func InstallGroup(name string) error {
    group, err := LoadGroup(name)
    if err != nil {
        return err
    }
    
    // Expand to package list
    return Install(group.Packages)
}
```

### Consequences

**Positive:**

- ✅ Easy onboarding (install @dev on new system)
- ✅ Curated package collections
- ✅ ShedOS-specific groups (@shedos-hyprland)
- ✅ Simple YAML format, community contributions easy

**Negative:**

- ❌ Must maintain group definitions
- ❌ Group membership changes over time

**Mitigation:**

- Groups versioned with shedman config packages
- Users can create custom groups in `~/.config/shedman/groups/`

### Alternatives Considered

1. **No groups**: Users manually install each package
2. **Meta-packages**: Require packaging, less flexible
3. **Shell scripts**: Not integrated, no management

---

## ADR-009: Simplified Command Set

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Original design had extensive command set including:

- `shedman aur` - AUR operations
- `shedman export` - Export package list
- `shedman import` - Import package list
- `shedman report` - Generate bug reports
- `shedman plugin` - Plugin management
- `shedman convert` - Package format conversion
- `shedman migrate` - Import pacman config

### Decision

**Remove unnecessary commands, keep essential:**

**✅ Keep:**

```bash
# Core package operations
shedman sync
shedman install <pkg>
shedman remove <pkg>
shedman update
shedman search <query>
shedman info <pkg>

# Utilities
shedman history
shedman doctor
shedman clean
shedman orphans
shedman owns <file>
shedman why <pkg>
shedman verify

# Groups
shedman group list
shedman group info @name

# Modules
shedman snapshot ...
shedman config ...
shedman de ...
shedman theme ...
shedman boot ...
shedman svc ...
shedman keyring ...
shedman security ...

# Building
shedman build PKGBUILD      # Build from PKGBUILD only
```

**❌ Remove:**

```bash
shedman aur          # AUR packages handled via main commands
shedman export       # Use: shedman -Qqe > pkglist.txt (pacman compat)
shedman import       # Use: shedman -S --needed - < pkglist.txt
shedman report       # Too specific, use standard bug report
shedman plugin       # No plugin system
shedman convert      # No format conversion
shedman migrate      # Read pacman.conf directly
shedman rollback     # No version management
```

### Consequences

**Positive:**

- ✅ Smaller command surface area
- ✅ Less documentation needed
- ✅ Easier to learn
- ✅ Pacman compatibility for export/import

**Negative:**

- ❌ Some convenience features lost

**Rationale for each removal:**

| Command | Why Removed | Alternative |
|---------|-------------|-------------|
| `aur` | AUR handled transparently | `shedman install <aur-pkg>` |
| `export` | pacman does this | `shedman -Qqe > pkglist` |
| `import` | pacman syntax works | `shedman -S - < pkglist` |
| `report` | Too specific | Manual bug reporting |
| `plugin` | No plugin system | Hooks for extensibility |
| `convert` | No cross-distro | Arch-only focused |
| `migrate` | Config read directly | Automatic |
| `rollback` | No version mgmt | System snapshots |

### Alternatives Considered

1. **Keep all commands**: Bloat, maintenance burden
2. **Remove more**: Risk losing useful features
3. **Aliases for removed**: Still maintain code

---

## ADR-010: No Plugin System (Initially)

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Original design included plugin system:

- JSON-RPC communication
- Plugin discovery
- Standalone executables
- Security considerations

**Reality:**

- No proven use cases yet
- Adds complexity
- Security surface area
- Maintenance burden

### Decision

**Do NOT implement plugin system in initial release.**

**Instead, use hooks for extensibility:**

```bash
# Hooks system (simple shell scripts)
/etc/shedman/hooks/
├── pre-install.d/
│   ├── 01-check-disk-space.sh
│   └── 50-backup-config.sh
├── post-install.d/
│   ├── 10-update-cache.sh
│   └── 90-notify-user.sh
├── pre-remove.d/
└── post-remove.d/
```

**Hook environment:**

```bash
#!/bin/bash
# Hooks receive environment variables
echo "Installing: $SHEDMAN_PACKAGE"
echo "Version: $SHEDMAN_VERSION"
echo "Action: $SHEDMAN_ACTION"
```

### Consequences

**Positive:**

- ✅ Simpler implementation
- ✅ No security concerns (hooks run as user)
- ✅ Familiar (like Git hooks)
- ✅ Easy to debug (just shell scripts)
- ✅ Faster development

**Negative:**

- ❌ Less powerful than plugins
- ❌ No isolation between hooks

**Future consideration:**

- If hooks prove insufficient, design plugin system later
- With real use cases, plugin API will be better designed

### Alternatives Considered

1. **Full plugin system**: Over-engineering without use cases
2. **No extensibility**: Too rigid
3. **Lua scripting**: Additional dependency, complexity

---

## ADR-011: Production-Ready, Not MVP

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

Two development philosophies:

1. **MVP (Minimum Viable Product)**:
   - Ship minimal feature set quickly
   - Iterate based on feedback
   - Accept technical debt

2. **Production-Ready**:
   - Feature-complete from day one
   - High quality, well-tested
   - Complete documentation
   - Ready for production use

### Decision

**Build production-ready, feature-complete product:**

**Quality standards:**

- ✅ Comprehensive test coverage
- ✅ All features documented
- ✅ Error handling for all edge cases
- ✅ Performance optimized
- ✅ Security hardened
- ✅ Man pages for all commands
- ✅ Shell completions (bash, zsh, fish)
- ✅ Example configurations
- ✅ Migration guides

**Not shipping until:**

- All planned modules complete
- Integration tests pass
- Real-world testing on ShedOS
- Documentation complete
- Performance benchmarks acceptable

### Consequences

**Positive:**

- ✅ Users get polished product
- ✅ Fewer bug reports
- ✅ Better reputation
- ✅ Less technical debt
- ✅ Easier maintenance

**Negative:**

- ❌ Longer time to first release
- ❌ Miss early feedback opportunity
- ❌ Risk over-engineering

**Trade-offs:**

- User expects it to "just work" on ShedOS
- shedman is core system component, must be reliable
- Better to launch late and great than early and broken

### Alternatives Considered

1. **MVP approach**: Risk users losing trust in early bugs
2. **Beta program**: Acceptable, can do closed beta
3. **Staged rollout**: Release modules incrementally

**Compromise:**

- Closed beta with early adopters
- Public release only when production-ready

---

## ADR-012: .shed Format for Future ShedOS Only

**Date**: 2026-01-11  
**Status**: Accepted  
**Deciders**: User (theshedman)

### Context

`.shed` format was designed as universal package format with:

- TUF security
- SLSA provenance
- Content-addressable storage
- zchunk delta updates
- Multi-architecture support
- Cross-distribution compatibility

**Reality:**

- Current ShedOS is Arch-based, uses pacman
- .shed format NOT needed for current implementation
- User plans to rebuild ShedOS from scratch in future
- .shed format is for that future rebuild

### Decision

**Document .shed format comprehensively BUT do NOT implement in current shedman.**

**Actions:**

1. ✅ Create complete .shed specification document
2. ✅ Save for future ShedOS rebuild from scratch
3. ❌ Do NOT implement in current codebase
4. ❌ Do NOT include /shed/ paths in shedrepo
5. ✅ Remove all .shed references from current shedman implementation

**Current shedman uses:**

- Native Arch packages (`.pkg.tar.zst`)
- pacman/go-alpm for management
- Standard Arch repository structure

### Consequences

**Positive:**

- ✅ Specification documented (won't forget design)
- ✅ Ready for future ShedOS rebuild
- ✅ Simpler current codebase (no .shed complexity)
- ✅ Focus on Arch package management
- ✅ Faster development

**Negative:**

- ❌ No universal package format now
- ❌ Specification might become outdated

**Mitigation:**

- Specification is versioned
- Can update when future ShedOS rebuild starts
- Current shedman provides solid foundation

### Alternatives Considered

1. **Implement .shed now**: Massive complexity, not needed
2. **Don't document .shed**: Lose design, hard to rebuild later
3. **Partial implementation**: Confusing, incomplete

---

## Summary Table

| ADR | Decision | Impact |
|-----|----------|--------|
| 001 | Arch-only | Focus, simplicity |
| 002 | Modular monorepo | Organization, cohesion |
| 003 | Capability interfaces | Flexibility, clarity |
| 004 | Pacman compatibility | User adoption |
| 005 | No version management | Aligns with Arch |
| 006 | go-alpm + pacman | Performance + reliability |
| 007 | Integrate existing tools | Maturity, less code |
| 008 | Package groups | User convenience |
| 009 | Simplified commands | Focused feature set |
| 010 | No plugins (yet) | Simplicity |
| 011 | Production-ready | Quality over speed |
| 012 | .shed for future only | Clear scope |

---

## Revision History

- **2026-01-11**: Initial ADRs based on architecture review and user decisions

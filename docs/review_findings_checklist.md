# Review Findings Checklist

This checklist tracks every fix identified in the review so we do not miss anything.

Legend: [ ] pending, [x] done

## Snapshot subsystem
- [x] Fix snapper list Target handling: local list when Target is nil; remote list via restic or rclone when Target is set. (`pkg/snapshot/backend_snapper.go`)
- [x] Use injected executor for snapper pull/push; remove duplicate dry-run check. (`pkg/snapshot/backend_snapper.go`)
- [x] Respect opts.DryRun and use injected executor for timeshift push/pull. (`pkg/snapshot/backend_timeshift.go`)
- [x] Fill scheduler status fields (NextRun/LastRun/Frequency) from systemd. (`pkg/snapshot/scheduler_systemd.go`)
- [x] Implement timeshift Diff or return an explicit unsupported error (no silent empty result). (`pkg/snapshot/backend_timeshift.go`)
- [x] Add/update tests for snapper list behaviors and timeshift/scheduler changes. (`pkg/snapshot/backend_snapper_test.go`, `pkg/snapshot/backend_timeshift_test.go`, new scheduler tests)

## ZFS backend
- [x] Add ZFS snapshot backend implementing Create/List/Delete/Restore/Diff. (new `pkg/snapshot/backend_zfs.go`)
- [x] Implement remote push/pull for ZFS (e.g., `zfs send` + transfer + `zfs receive`). (`pkg/snapshot/backend_zfs.go`)
- [x] Wire ZFS into factory detection and backend creation. (`pkg/snapshot/factory.go`)
- [x] Add tests for ZFS backend detection and basic operations. (new tests under `pkg/snapshot/`)
- [x] Track datasets by `com.shedman:id` and apply Delete/Restore/Push/Pull/Diff across matching snapshots; add multi-dataset tests. (`pkg/snapshot/backend_zfs.go`, `pkg/snapshot/backend_zfs_test.go`)
- [x] Document prerequisites/behavior for ZFS snapshots (zfs, sudo, datasets). (`docs/`)

## AUR options
- [x] Honor `FetchPGPKeys` and `SkipPGPCheck` in `InstallFull`. (`pkg/core/aur.go`)
- [x] Ensure makepkg invocation supports `--skippgpcheck` when requested. (`pkg/core/aur.go`)
- [x] Define sandbox GPG key behavior (GNUPGHOME/binds) so fetched keys are visible in bwrap builds. (`pkg/core/aur.go`)
- [x] Add tests for option wiring and command args. (`pkg/core/aur_test.go`)

## Core error semantics
- [x] RetryClient returns a clear error when mirror list is empty. (`internal/http/client.go`)
- [x] Engine.SearchFiles returns explicit error when unsupported. (`pkg/core/engine.go`)
- [x] MultiSourceResolver returns explicit error for forced source missing and surfaces source errors when appropriate. (`pkg/core/multi_source.go`)
- [x] Add tests for the updated error behavior. (`internal/http/client_test.go`, `pkg/core/multi_source_test.go`)

## Stub/basic implementations
- [x] Security scanner: implement real vulnerability retrieval via engine/backend and convert to Vulnerability structs. (`pkg/security/manager.go`)
- [x] Keyring manager: implement List/Add/Remove using system keyring tools; define error handling. (`pkg/keyring/manager.go`)
- [x] Mirror reflector: implement List and Test (parse mirrorlist and/or reflector output). (`pkg/mirror/reflector.go`)
- [x] Boot manager: determine current kernel; implement GRUB support path. (`pkg/boot/manager.go`)
- [x] Add tests for these implementations. (`pkg/security/manager_test.go`, `pkg/keyring/manager_test.go`, `pkg/mirror/manager_test.go`, `pkg/boot/manager_test.go`)

## Docs/regression
- [x] Update docs for new behaviors (AUR PGP policy, ZFS support, snapshot remote behavior). (`docs/`)
- [x] Run targeted unit tests for touched packages and address regressions. (`go test ./...` with focus on changed packages)

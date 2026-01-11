# Repository Restructuring File Mapping

## Files Moved

### pkg/core/ (42 files)
- engine.go, engine_test.go, backend.go (3 files)
- resolver/ (16 files): deps, conflicts, groups, multi_source, removal, summary, version + tests
- installer/ (14 files): installer, transaction, aur, shedos, shed, cache, progress + tests
- pkgdb/ (8 files): pkgdb, pacman_db, aur_db, shed_db + tests

### pkg/backend/ (22 files)
- backend.go, options.go, errors.go, errors_test.go, registry.go, detect.go, detect_test.go (7 files)
- pacman/ (11 files): alpm.go, alpm_backend.go, pacman.go, pacman_conf.go, db.go, files.go, progress.go + tests
- aur/ (2 files): aur.go, aur_test.go
- shedrepo/ (2 files): shedrepo.go, shedrepo_test.go

### internal/ (17 files)
- config/ (3 files): config.go, config_test.go, types.go
- output/ (8 files): color.go, help.go, info.go, progress.go, prompt.go, table.go + tests
- cache/ (2 files): cache.go, cache_test.go
- http/ (2 files): client.go, client_test.go
- signals/ (2 files): signals.go, signals_test.go

### cmd/shedman/ (15 files)
- main.go (root command)
- commands/ (14 files): install, remove, search, update, info, sync, version + tests

## Files Remaining (To Delete Later)
- pkg/shedman/backend/{apt,dnf,zypper}/ - cross-distro backends
- pkg/shedman/convert/ - package conversion
- pkg/shedman/migrate/ - migration utilities
- cmd/{migrate,rollback}.go - deprecated commands

## Total: 96 files moved
All moves performed with git mv to preserve history.
All files compile (with expected import errors to fix next).

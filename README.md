<p align="center">
  <h1 align="center">shedman</h1>
  <p align="center">
    <strong>A next-generation universal package manager for ShedOS and beyond</strong>
  </p>
  <p align="center">
    <a href="#features">Features</a> •
    <a href="#installation">Installation</a> •
    <a href="#usage">Usage</a> •
    <a href="#documentation">Documentation</a> •
    <a href="#contributing">Contributing</a> •
    <a href="#license">License</a>
  </p>
</p>

---

## Overview

**shedman** is a modern package manager designed to be a drop-in replacement for pacman on Arch-based systems, while also providing cross-distribution package management capabilities. It aims to solve Linux's packaging fragmentation by providing a unified interface that works everywhere.

> *"You should never have to think twice before reformatting your computer."*

## Features

- **100% Pacman Compatible** — Drop-in replacement for all pacman commands and flags
- **AUR Integration** — Seamless access to the Arch User Repository with sandboxed builds
- **Universal `.shed` Packages** — Install packages from any Linux distribution
- **System Snapshots** — Backup and restore your entire system state (packages, configs, themes)
- **Cloud Sync** — Push snapshots to Google Drive, Cloudflare R2, AWS S3, or USB drives
- **Desktop Environment Switching** — Switch between Hyprland, GNOME, KDE, COSMIC with one command
- **Configuration Management** — Apply, diff, and rollback application configurations
- **Security First** — Ed25519 signing, TUF metadata, CVE checking, sandboxed AUR builds

## Installation

### From Source (Recommended for Development)

```bash
# Clone the repository
git clone https://github.com/theshedman/shedman.git
cd shedman

# Build
go build -o shedman .

# Run
./shedman --help
```

### On ShedOS

shedman comes pre-installed on ShedOS.

### From Go

```bash
go install github.com/theshedman/shedman@latest
```

## Usage

### Basic Commands

```bash
# Sync package databases
shedman sync

# Install a package
shedman install neovim

# Install from AUR
shedman install neovim-nightly --aur

# Remove a package
shedman remove firefox

# Search for packages
shedman search vim

# Update all packages
shedman update
```

### Pacman Compatibility

All pacman commands work exactly the same:

```bash
shedman -Syu                    # Update system
shedman -S neovim               # Install package
shedman -R firefox              # Remove package
shedman -Ss vim                 # Search packages
shedman -Qi neovim              # Query package info
```

### Snapshots

```bash
# Create a snapshot
shedman snapshot create --name "before-update"

# List snapshots
shedman snapshot list

# Restore a snapshot
shedman snapshot restore 5

# Push to cloud
shedman snapshot push 5 --remote gdrive
```

### Desktop Environment Switching

```bash
# List available DEs
shedman de list

# Switch to GNOME
shedman de switch gnome
```

## Documentation

For comprehensive documentation, see:

- **[Implementation Plan](docs/README.md)** — Complete architecture and CLI reference
- **[Contributing Guide](CONTRIBUTING.md)** — How to contribute (coming soon)

## Project Status

shedman is currently in early development. See the [implementation phases](docs/README.md#part-3-implementation-phases) for the roadmap.

| Phase | Status | Description |
|-------|--------|-------------|
| Core Commands | 🚧 In Progress | sync, install, remove, search |
| Repository | 📋 Planned | shedrepo, R2 integration, CI/CD |
| TUI | 📋 Planned | Interactive terminal UI |
| Snapshots | 📋 Planned | Local and cloud backups |

## Requirements

- **Go** 1.21 or later
- **Linux** (Arch-based recommended, cross-distro support planned)

## Contributing

We welcome contributions! Here's how you can help:

### Getting Started

1. **Fork** the repository
2. **Clone** your fork:

   ```bash
   git clone https://github.com/YOUR_USERNAME/shedman.git
   ```

3. **Create a branch** for your feature:

   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make your changes** following our coding standards
5. **Write tests** (we follow TDD — tests first!)
6. **Run tests**:

   ```bash
   go test ./... -v
   ```

7. **Commit** with a descriptive message:

   ```bash
   git commit -m "feat: add support for X"
   ```

8. **Push** and create a Pull Request

### Development Guidelines

- **Test-Driven Development (TDD)**: Write failing tests first, then implement
- **Go Idioms**: Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- **Commit Messages**: Use [Conventional Commits](https://www.conventionalcommits.org/) format
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation
  - `test:` for tests
  - `refactor:` for refactoring

### Code of Conduct

Please be respectful and inclusive. We follow the [Contributor Covenant](https://www.contributor-covenant.org/).

### Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Include reproduction steps for bugs
- Check existing issues before creating new ones

## Architecture

```
shedman/
├── cmd/                    # CLI commands (Cobra)
├── pkg/                    # Public packages (library API)
│   ├── backend/            # Package backends (pacman, aur, shedrepo)
│   ├── config/             # Configuration management
│   └── snapshot/           # Snapshot functionality
├── internal/               # Private packages
├── docs/                   # Documentation
└── main.go                 # Entry point
```

## Related Projects

- **[ShedOS](https://github.com/theshedman/shedos)** — The Arch-based Linux distribution
- **[shedrepo](https://github.com/theshedman/shedrepo)** — Package repository and build scripts

## License

This project is licensed under the **GNU General Public License v3.0** — see the [LICENSE](LICENSE) file for details.

---

<p align="center">
  Made with ❤️ for the Linux community
</p>

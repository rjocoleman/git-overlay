# git-overlay

Git Overlay is a tool for managing overlay repositories that extend or modify upstream Git repositories. It's an alternative to maintaining a fork that facilitates adding custom modifications to upstream code while making it easy to track upstream changes and provide a clean development environment.

## Installation

### Binary

Download the latest binary from the [releases page](https://github.com/rjocoleman/git-overlay/releases).

### Using Homebrew

```bash
# Install git-overlay
brew install rjocoleman/git-overlay/git-overlay
```

### From Source

```bash
go install github.com/rjocoleman/git-overlay@latest
```

## Usage

### Initialize a New Overlay

```bash
# Create configuration file
cat > .git-overlay.yml << EOF
upstream:
  url: "https://github.com/example/repo.git"
  ref: "main"
symlinks:
  - app
  - config
  - from: src/lib
    to: library
EOF

# Initialize overlay repository
git-overlay init
```

### Update Upstream Code

```bash
# Update upstream code and rebuild links
git-overlay sync

# Force update (overwrite existing files)
git-overlay sync --force
```

### Clean Managed Files

```bash
# Remove managed files and links from the overlay directory
git-overlay clean
```

This command removes:
- Files and links that are managed by git-overlay (configured in `.git-overlay.yml`)
- Empty directories that contained only managed files
- Empty parent directories after managed files are removed

Custom files and directories in the overlay directory are preserved.

This is useful when you want to:
- Remove managed files before updating configuration
- Clean up stale links and empty directories
- Prepare for syncing with a different upstream version

Example:
```
overlay/                 # Before cleaning
├── custom.txt          # Custom file
├── managed.txt         # Managed symlink
└── dir/
    ├── empty/          # Empty directory
    ├── keep/           # Directory with custom files
    │   └── custom.txt  # Custom file
    └── managed.txt     # Managed symlink

overlay/                 # After cleaning
├── custom.txt          # Preserved
└── dir/                # Preserved (has content)
    └── keep/           # Preserved
        └── custom.txt  # Preserved
```

### Configuration

The tool uses a YAML configuration file (default: `.git-overlay.yml`):

```yaml
upstream:
  url: "https://github.com/example/repo.git"  # Upstream repository URL
  ref: "main"                                 # Branch, tag, or commit to track

symlinks:                      # Files/directories to link from upstream
  - app                        # Simple form: same path for source and target
  - config                     # Will link .upstream/config to overlay/config
  - from: src/lib              # Extended form: custom target path
    to: library                # Will link .upstream/src/lib to overlay/library
```

### Link Modes

- `symlink` (default): Creates symbolic links
- `hardlink`: Creates hard links (files only)
- `copy`: Creates copies of files/directories

```bash
# Use different link mode
git-overlay sync --link-mode hardlink
```

### Global Flags

- `-c, --config <path>`: Path to config file (default: `.git-overlay.yml`)
- `-f, --force`: Force overwrite of existing files/links
- `--link-mode <mode>`: Link mode (symlink|hardlink|copy)
- `--debug`: Enable debug logging

## Project Structure

```
project-root/
├── .git/                       # Main repository Git data
├── .upstream/                  # Upstream repository files
│   ├── .git/                   # Upstream Git data (ignored)
│   └── [upstream files]        # Upstream source files
├── .git-overlay.yml            # Overlay configuration
├── .gitignore                  # Local Git ignore rules
├── .gitignore-overlay          # Tracked overlay ignore rules
└── overlay/                    # Overlay working directory
    ├── [custom files]          # Custom overlay files
    └── [linked files]          # Links to upstream files
```

## Development

Requirements:
- Go 1.23 or later

```bash
# Clone repository
git clone https://github.com/rjocoleman/git-overlay.git
cd git-overlay

# Build
go build

# Install locally
go install
```

## Troubleshooting

### Common Issues

1. **Symlink creation fails**
   - Check if your system allows symlinks
   - Try using `--link-mode hardlink` or `--link-mode copy` instead
   - Use `--force` if target already exists

2. **Upstream sync fails**
   - Ensure you have access to the upstream repository
   - Check if the specified ref (branch/tag) exists
   - Configure Git to allow file protocol: `git config --global protocol.file.allow always`

3. **Path validation errors**
   - Ensure symlink targets don't try to escape the overlay directory
   - Avoid absolute paths in configuration
   - Use relative paths from the repository root

### Common Workflows

1. **Adding new files from upstream**
   ```bash
   # Add new files to .git-overlay.yml
   git-overlay sync --force
   git add .git-overlay.yml overlay/
   git commit -m "Add new files from upstream"
   ```

2. **Updating upstream version**
   ```bash
   # Update ref in .git-overlay.yml
   git-overlay sync
   git add .git-overlay.yml .upstream
   git commit -m "Update upstream to new version"
   ```

3. **Cleaning and rebuilding**
   ```bash
   # Clean everything and rebuild
   git-overlay clean
   git-overlay sync --force
   ```

4. **Working with dotfiles**
   ```yaml
   # .git-overlay.yml
   symlinks:
     - .eslintrc
     - .config
     - from: .github/workflows
       to: .github/workflows
   ```

## License

MIT License

# ViewNet

**Network discovery and port scanning tool with an interactive TUI**

## Usage

```bash
# Interactive mode (default)
viewnet

# Help
viewnet -h

# IP discovery only
viewnet -ips

# Port scan specific ports
viewnet -p 22,80,443

# Export to CSV
viewnet -csv results.csv

# Non-interactive CSV export
viewnet -ips -csv scan.csv
```

## Features

- **Auto-discovery**: Detects local subnet automatically
- **Interactive TUI**: Real-time results with search/filter (`/` or `f`)
- **Vendor detection**: Identifies device manufacturers via MAC addresses
- **Export**: CSV output for further analysis
- **Cross-platform**: Windows, Linux

## Search & Filter

- Press `/` or `f` to search
- Search by IP, hostname, vendor, MAC, or services
- `Ctrl+F` for focused search (IP/vendor only)
- `r` to rescan, `q` to quit

## Build

```bash
# Install dependencies
go mod download

# Build
go build -o viewnet .

# Or use justfile
just build

# Build for Linux and Windows
just build-all
```

## Releases

Binaries are automatically built and released for Linux and Windows when a new git tag is pushed:

```bash
# Create and push a new release
git tag v1.0.0
git push origin v1.0.0
```

The GitHub Actions workflow will:
- Build `viewnet-linux-amd64` and `viewnet-windows-amd64.exe`
- Create a GitHub release
- Upload binaries as release assets

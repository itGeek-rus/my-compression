# my-compression

Local web app to archive or extract a file without data loss, show sizes and progress, then download the result.

**Pack:** ZIP, TAR.GZ, ZSTD, TAR.XZ  
**Unpack:** the same, plus 7Z

## Run from source

```bash
go test ./...
go run ./cmd/app
```

The app opens http://127.0.0.1:9005 in the browser. Stop it with **Stop application** on the page, or `Ctrl+C` in the terminal.

```bash
go build -o my-compression ./cmd/app
./my-compression
```

## Config

| Env | Default | Meaning |
|---|---|---|
| `ADDR` | `127.0.0.1:9005` | Listen address |
| `MAX_UPLOAD_BYTES` | `33554432` (32 MiB) | Upload limit |
| `TEMP_DIR` | OS temp | Working directories for jobs |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown |

## Install from GitHub Releases

After a version tag is pushed, installers appear on the repository **Releases** page.

| OS | File | How to install |
|---|---|---|
| Windows | `MyCompression-Setup-windows-amd64.exe` | Run the installer, then open the shortcut |
| macOS Apple Silicon | `MyCompression-macos-arm64.dmg` | Open the disk image and drag **My Compression** to Applications |
| macOS Intel | `MyCompression-macos-amd64.dmg` | Same as above |
| Debian / Ubuntu | `my-compression_*_amd64.deb` | `sudo dpkg -i my-compression_*_amd64.deb` |

The first launch on macOS may require **Right-click → Open** (Gatekeeper), or:

```bash
xattr -cr "/Applications/My Compression.app"
```

Windows SmartScreen may ask to run an unknown publisher installer anyway.

## Publish a release

```bash
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions builds Windows, macOS, and Linux packages and attaches them to the release.

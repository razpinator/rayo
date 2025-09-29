# Rayo Distribution Guide

This guide covers how to distribute the Rayo programming language across different platforms and environments.

## Overview

Rayo is distributed as a single binary compiler that transpiles Rayo source code to Go. Since Rayo leverages Go's ecosystem, distribution is straightforward and leverages Go's excellent cross-compilation capabilities.

## Supported Platforms

Rayo can be built for the following platforms:

| OS       | Architecture | Target            |
|----------|--------------|-------------------|
| Linux    | x86_64       | linux-amd64       |
| Linux    | ARM64        | linux-arm64       |
| macOS    | x86_64       | darwin-amd64      |
| macOS    | ARM64 (M1+)  | darwin-arm64      |
| Windows  | x86_64       | windows-amd64     |
| Windows  | ARM64        | windows-arm64     |

## Distribution Methods

### 1. GitHub Releases (Recommended)

Use GitHub releases for official distributions:

```bash
# Build all platforms
make dist-all

# Create release archives
for binary in build/*; do
    if [[ $binary == *.exe ]]; then
        zip -j "$binary.zip" "$binary"
    else
        tar -czf "$binary.tar.gz" -C build "$(basename $binary)"
    fi
done
```

**Release Structure:**
```
rayo-v1.0.0/
├── rayoc-linux-amd64.tar.gz
├── rayoc-linux-arm64.tar.gz
├── rayoc-darwin-amd64.tar.gz
├── rayoc-darwin-arm64.tar.gz
├── rayoc-windows-amd64.zip
├── rayoc-windows-arm64.zip
├── README.md
└── CHANGELOG.md
```

### 2. Package Managers

#### Homebrew (macOS/Linux)
```ruby
# Formula/rayo.rb
class Rayo < Formula
  desc "Rayo programming language"
  homepage "https://github.com/razpinator/rayo"
  url "https://github.com/razpinator/rayo/releases/download/v1.0.0/rayoc-darwin-amd64.tar.gz"
  sha256 "YOUR_SHA256_HERE"

  def install
    bin.install "rayoc"
  end

  test do
    (testpath/"hello.ryo").write 'print("Hello, World!")'
    system "#{bin}/rayoc", "transpile", "hello.ryo"
  end
end
```

#### Chocolatey (Windows)
```xml
<!-- rayo.nuspec -->
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2015/06/nuspec.xsd">
  <metadata>
    <id>rayo</id>
    <version>1.0.0</version>
    <title>Rayo Programming Language</title>
    <authors>Razvan</authors>
    <description>Rayo is a programming language that transpiles to Go</description>
  </metadata>
</package>
```

#### Snap (Linux)
```yaml
# snapcraft.yaml
name: rayo
version: '1.0.0'
summary: Rayo programming language
description: A programming language that transpiles to Go

grade: stable
confinement: strict

apps:
  rayoc:
    command: rayoc

parts:
  rayo:
    plugin: dump
    source: https://github.com/razpinator/rayo/releases/download/v1.0.0/rayoc-linux-amd64.tar.gz
    organize:
      rayoc: bin/rayoc
```

### 3. Docker Distribution

```dockerfile
# Dockerfile
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY rayoc-linux-amd64 /usr/local/bin/rayoc
ENTRYPOINT ["/usr/local/bin/rayoc"]
```

```bash
# Build and push
docker build -t razpinator/rayo:latest .
docker push razpinator/rayo:latest
```

### 4. Direct Downloads

Host binaries on a CDN or web server:

```bash
# Simple HTTP server
python3 -m http.server 8000

# Or use a cloud storage service
aws s3 cp build/ s3://rayo-releases/ --recursive
```

## Installation Instructions

### macOS
```bash
# Using Homebrew
brew install razpinator/tap/rayo

# Or manual download
curl -L https://github.com/razpinator/rayo/releases/download/v1.0.0/rayoc-darwin-amd64.tar.gz | tar xz
sudo mv rayoc /usr/local/bin/
```

### Linux
```bash
# Using Snap
sudo snap install rayo --edge

# Or manual download
wget https://github.com/razpinator/rayo/releases/download/v1.0.0/rayoc-linux-amd64.tar.gz
tar xzf rayoc-linux-amd64.tar.gz
sudo mv rayoc /usr/local/bin/
```

### Windows
```bash
# Using Chocolatey
choco install rayo

# Or manual download
# Download rayoc-windows-amd64.zip from releases
# Extract and add to PATH
```

## CI/CD Pipeline

### GitHub Actions Example

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'

      - name: Build all platforms
        run: make dist-all

      - name: Create release archives
        run: |
          cd build
          for file in rayoc-*; do
            if [[ $file == *.exe ]]; then
              zip -j "$file.zip" "$file"
            else
              tar -czf "$file.tar.gz" "$file"
            fi
          done

      - name: Create GitHub release
        uses: softprops/action-gh-release@v1
        with:
          files: build/*.tar.gz
          files: build/*.zip
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Runtime Dependencies

Rayo has minimal runtime dependencies:

- **Compiler**: No dependencies (statically linked)
- **Generated Go code**: Requires Go runtime for execution
- **Standard library**: Uses only standard Go libraries

## Versioning Strategy

Use semantic versioning:
- **Major**: Breaking changes to language syntax/semantics
- **Minor**: New features, backward compatible
- **Patch**: Bug fixes, no API changes

## Distribution Checklist

- [ ] Build binaries for all supported platforms
- [ ] Create release archives with proper naming
- [ ] Generate checksums for verification
- [ ] Update package manager formulas
- [ ] Test installation on each platform
- [ ] Update documentation with new version
- [ ] Announce release on relevant channels

## Security Considerations

- Sign releases with GPG keys
- Provide checksums (SHA256) for verification
- Use HTTPS for all distribution channels
- Regularly update dependencies
- Consider code signing for Windows binaries

## Performance Optimization

For faster distribution:
- Use UPX to compress binaries
- Strip debug symbols for release builds
- Consider different optimization levels
- Use CDN for global distribution

```bash
# Compress binaries
upx --best build/rayoc-*

# Strip symbols
go build -ldflags="-s -w" ./cmd/rayoc
```</content>
<parameter name="filePath">/Users/razmax/Documents/dev/rayo/DISTRIBUTION.md

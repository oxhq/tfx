# Installing TFX

The TFX release pipeline is configured to publish assets for:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`
- `windows/arm64`

## Quick Install

### Unix-like systems

```bash
curl -fsSL https://raw.githubusercontent.com/oxhq/tfx/main/tools/install.sh | bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/oxhq/tfx/main/tools/install.sh | bash -s -- -v v0.2.0
```

Change the install directory:

```bash
curl -fsSL https://raw.githubusercontent.com/oxhq/tfx/main/tools/install.sh | bash -s -- -d /usr/local/bin
```

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/oxhq/tfx/main/tools/install.ps1 | iex
```

Install a specific version:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/oxhq/tfx/main/tools/install.ps1))) -Version v0.2.0
```

## Manual Install

1. Download the asset for your platform from the GitHub releases page.
2. Extract the archive.
3. Move `tfx` or `tfx.exe` into a directory on your `PATH`.
4. Verify the binary:

```bash
tfx --version
```

## Local Developer Install

When working from source:

```bash
make install-local
```

By default this installs `tfx` into `~/.local/bin`. Override the prefix:

```bash
make install-local PREFIX=/usr/local
```

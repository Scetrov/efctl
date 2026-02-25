# efctl

![efctl Logo](https://raw.githubusercontent.com/Scetrov/efctl/refs/heads/main/assets/logo.png)

`efctl` is a fast and flexible CLI designed to automate the setup, deployment, and teardown of the EVE Frontier local world contracts and smart gates. Built with Go, it provides an intuitive interface to seamlessly initialize the Sui playground environment and interact with the local blockchain.

## Features

- **Automated setup**: Fetches and prepares the local EVE Frontier environment in seconds.
- **Smart Gate lifecycle**: Supports `up` and `down` commands to gracefully manage container lifecycles.
- **GraphQL query tools**: Interact dynamically with the local Sui GraphQL RPC to query objects and packages.
- **Dependency validation**: Quickly verifies mandatory local prerequisites (like Docker/Podman, Git, and open ports).

## Installation

Below are one-liner installers for each platform. They automatically detect your CPU architecture (`amd64` / `arm64`) and download the correct binary from [GitHub Releases](https://github.com/Scetrov/efctl/releases).

> **Tip:** Replace `latest` with a specific tag (e.g. `v0.1.0`) to pin a version.

---

### Linux

#### curl

```bash
curl -fsSL https://github.com/Scetrov/efctl/releases/latest/download/efctl-linux-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') \
  -o /tmp/efctl && chmod +x /tmp/efctl && sudo mv /tmp/efctl /usr/local/bin/efctl
```

#### wget

```bash
wget -qO /tmp/efctl https://github.com/Scetrov/efctl/releases/latest/download/efctl-linux-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') \
  && chmod +x /tmp/efctl && sudo mv /tmp/efctl /usr/local/bin/efctl
```

---

### macOS

#### curl (Homebrew not required)

```bash
curl -fsSL https://github.com/Scetrov/efctl/releases/latest/download/efctl-darwin-$(uname -m | sed 's/x86_64/amd64/;s/arm64/arm64/') \
  -o /tmp/efctl && chmod +x /tmp/efctl && sudo mv /tmp/efctl /usr/local/bin/efctl
```

---

### Windows (PowerShell)

```powershell
iex ((New-Object Net.WebClient).DownloadString('data:text/plain,') + @'
$arch = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else { "amd64" }
$url = "https://github.com/Scetrov/efctl/releases/latest/download/efctl-windows-$arch.exe"
$dest = "$env:LOCALAPPDATA\efctl\efctl.exe"
New-Item -ItemType Directory -Force -Path (Split-Path $dest) | Out-Null
Invoke-WebRequest -Uri $url -OutFile $dest
$path = [Environment]::GetEnvironmentVariable("Path", "User")
if ($path -notlike "*$(Split-Path $dest)*") {
    [Environment]::SetEnvironmentVariable("Path", "$path;$(Split-Path $dest)", "User")
}
Write-Host "efctl installed to $dest — restart your terminal to use it."
'@)
```

> **Note:** This installs to `%LOCALAPPDATA%\efctl` and adds it to your user `PATH`. You may need to restart your terminal for the `PATH` change to take effect.

---

### From Source

Ensure you have [Go 1.25+](https://go.dev/dl/) installed.

```bash
git clone https://github.com/Scetrov/efctl.git
cd efctl
go build -trimpath -ldflags="-s -w" -o efctl main.go
sudo mv efctl /usr/local/bin/   # Linux/macOS
# Windows: move efctl.exe to a directory in your PATH
```

## Quick Start

To spin up your local environment:

```bash
efctl env up
```

For more detailed usage instructions, check out the [USAGE.md](USAGE.md) file.

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for more details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

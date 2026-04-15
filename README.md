# ⚡ efctl

![efctl Logo](https://raw.githubusercontent.com/Scetrov/efctl/refs/heads/main/assets/logo.png)

> [!IMPORTANT]
> `efctl` is a high-performance CLI designed to automate the lifecycle of EVE Frontier local world contracts and smart gates.

Built with **Go**, it provides a premium interface to seamlessly initialize the Sui playground environment and interact with the local blockchain.

## Features

- **Automated setup**: Fetches and prepares the local EVE Frontier environment in seconds.
- **Smart Gate lifecycle**: Supports `up` and `down` commands to gracefully manage container lifecycles.
- **Builder Flow automation**: Includes `extension` and `run` commands to quickly initialize, publish, and interact with extensions in the builder-scaffold container.
- **GraphQL query tools**: Interact dynamically with the local Sui GraphQL RPC to query objects and packages.
- **Dependency validation**: Quickly verifies mandatory local prerequisites (like Docker/Podman, Git, and open ports).

## Installation

Below are one-liner installers for each platform. They automatically detect your CPU architecture (`amd64` / `arm64`) and download the correct binary from [GitHub Releases](https://github.com/Scetrov/efctl/releases).

> [!TIP]
> Replace `latest` with a specific tag (e.g. `v0.1.0`) to pin a version.

---

### Linux

#### curl

```bash
cd ~ && curl -fsSL https://github.com/Scetrov/efctl/releases/latest/download/efctl-linux-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') \
  -o /tmp/efctl && chmod +x /tmp/efctl && mkdir -p ~/.local/bin && mv /tmp/efctl ~/.local/bin/efctl
```

#### wget

```bash
cd ~ && wget -qO /tmp/efctl https://github.com/Scetrov/efctl/releases/latest/download/efctl-linux-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') \
  && chmod +x /tmp/efctl && mkdir -p ~/.local/bin && mv /tmp/efctl ~/.local/bin/efctl
```

---

### macOS

#### curl (Homebrew not required)

```zsh
cd ~ && curl -fsSL "https://github.com/Scetrov/efctl/releases/latest/download/efctl-darwin-$(uname -m | sed 's/x86_64/amd64/;s/arm64/arm64/')" \
  -o /tmp/efctl && chmod +x /tmp/efctl && mkdir -p ~/.local/bin && mv /tmp/efctl ~/.local/bin/efctl
```

---

### Windows (PowerShell)

```powershell
& {
  $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
  if ($isAdmin){
    throw "Please run this script as a non-administrator."
  }
  cd ~
  $arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
  $url = "https://github.com/Scetrov/efctl/releases/latest/download/efctl-windows-$arch.exe"
  $dest = Join-Path $HOME "bin\scetrov\efctl\efctl.exe"
  New-Item -ItemType Directory -Force -Path (Split-Path $dest) | Out-Null
  Invoke-WebRequest -Uri $url -OutFile $dest
  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ($userPath -notlike "*$(Split-Path $dest)*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$(Split-Path $dest)", "User")
  }
  Write-Host "efctl installed to $dest — restart your terminal to use it."
}
```

> [!NOTE]
> This installs to `~/bin/scetrov/efctl` and adds it to your user `PATH`. You may need to restart your terminal for the `PATH` change to take effect.

---

### From Source

Ensure you have [Go 1.26.2+](https://go.dev/dl/) installed.

```bash
git clone https://github.com/Scetrov/efctl.git
cd efctl
go build -trimpath -ldflags="-s -w" -o efctl main.go
mkdir -p ~/.local/bin && mv efctl ~/.local/bin/   # Linux/macOS
# Windows: move efctl.exe to a directory in your PATH
```

> [!NOTE]
> Make sure `~/.local/bin` is in your `PATH`. Add `export PATH="$HOME/.local/bin:$PATH"` to your shell profile if needed.

## Quick Start

To spin up your local environment:

```bash
mkdir -p ~/ef
cd ~/ef
efctl env up
```

![efctl demo](https://raw.githubusercontent.com/Scetrov/efctl/main/assets/efctl_2026_02_opt.gif)

## 📚 Documentation

Detailed documentation is available in the repository:

- **[Usage Guide](USAGE.md)**: Comprehensive command reference and configuration guides.
- **[CLI Reference](docs/efctl.md)**: Auto-generated command-line documentation.
- **[Contributing](CONTRIBUTING.md)**: Guidelines for contributing to the project.
- **[Security](SECURITY.md)**: Security policy and reporting.

---

Add the output directory to your PATH:

```bash
export PATH=$PWD/output:$PATH
```

Build the project:

```bash
make build
```

Run tests:

```bash
make test
```

## AI Tooling

This repository includes project-local context-mode configuration for both Gemini CLI and VS Code Copilot.

- Gemini CLI reads [.gemini/settings.json](.gemini/settings.json) and [GEMINI.md](GEMINI.md).
- VS Code Copilot reads [.vscode/mcp.json](.vscode/mcp.json), [.github/hooks/context-mode.json](.github/hooks/context-mode.json), and [.github/copilot-instructions.md](.github/copilot-instructions.md).

Machine-level prerequisite:

```bash
npm install -g context-mode
```

After installing, restart Gemini CLI and VS Code so both clients pick up the MCP server and hooks.

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for more details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

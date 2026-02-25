# efctl

![efctl Logo](./assets/efctl-logo.png)

`efctl` is a fast and flexible CLI designed to automate the setup, deployment, and teardown of the EVE Frontier local world contracts and smart gates. Built with Go, it provides an intuitive interface to seamlessly initialize the Sui playground environment and interact with the local blockchain.

## Features

- **Automated setup**: Fetches and prepares the local EVE Frontier environment in seconds.
- **Smart Gate lifecycle**: Supports `up` and `down` commands to gracefully manage container lifecycles.
- **GraphQL query tools**: Interact dynamically with the local Sui GraphQL RPC to query objects and packages.
- **Dependency validation**: Quickly verifies mandatory local prerequisites (like Docker/Podman, Git, and open ports).

## Installation

### From Source

Ensure you have Go installed on your machine (1.20+ recommended).

```bash
git clone https://github.com/your-org/efctl.git
cd efctl
go build -o efctl main.go
sudo mv efctl /usr/local/bin/
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

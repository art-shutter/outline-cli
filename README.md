# Outline CLI

A command-line interface for managing Outline servers. This CLI allows you to store and manage multiple Outline server configurations locally and perform basic operations on them.
Inspired by https://github.com/JMCFTW/outline-cli/

## Installation

### Build from source

```bash
git clone github.com/art-shutter/outline-cli
cd outline-cli
just build
```

### Using nix flake

```bash
nix shell 'github:art-shutter/outline-cli/main#outline-cli'
```

## Configuration

The CLI stores configuration in `~/.config/outline-cli/config.yaml`. The configuration file is automatically created when you add your first server.

Example configuration:
```yaml
servers:
  my-server:
    name: my-server
    url: https://myserver.com/SecretPath
  production:
    name: production
    url: https://prod-server.com/AnotherSecretPath
```

## Usage

### Server Management

#### List all servers
```bash
outline-cli servers list
```

#### Add a new server
```bash
outline-cli servers add <server-name> <server-url>
```

Example:
```bash
outline-cli servers add my-server https://myserver.com/SecretPath
```

#### Get server details
```bash
outline-cli servers get <server-name>
```

#### Update server URL
```bash
outline-cli servers update <server-name> --url <new-url>
```

#### Delete a server
```bash
outline-cli servers delete <server-name>
```

### Access Key Management

#### List access keys for a server
```bash
outline-cli servers keys list <server-name>
```

#### Create a new access key
```bash
outline-cli servers keys create <server-name> [--name <key-name>] [--method <encryption-method>] [--port <port>]
```

Example:
```bash
outline-cli servers keys create my-server --name "My Key" --method aes-192-gcm --port 12345
```

#### Delete an access key
```bash
outline-cli servers keys delete <server-name> <key-id>
```

### Server Metrics

#### View transfer metrics
```bash
outline-cli servers metrics <server-name>
```

## Help

Get help for any command:
```bash
outline-cli --help
outline-cli servers --help
outline-cli servers add --help
```

## Development

### Project Structure

```
outline-cli/
├── main.go          # Main CLI entry point and argument parsing
├── config.go        # Configuration management and CRUD operations
├── api.go           # Outline server API client
├── go.mod           # Go module definition
├── justfile         # Task runner for building and development
└── README.md        # This file
```

### Building

```bash
just build
```

### Running tests

```bash
just test
```

### Available Commands

The project uses `just` as a task runner. Available commands:

```bash
just build    # Build the CLI binary
just clean    # Remove build artifacts
just test     # Run tests
just install  # Install globally
just fmt      # Format code
just lint     # Run linter
just dev      # Build and test CLI functionality
just help     # Show all available commands
```

## License

This project is licensed under the MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

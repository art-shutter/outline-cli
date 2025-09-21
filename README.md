# Outline CLI

A command-line interface for managing Outline servers. This CLI allows you to store and manage multiple Outline server configurations locally and perform basic operations on them.
Inspired by https://github.com/JMCFTW/outline-cli/  
Motivation: the official outline Electron app is unreliable.

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

### Installing binaries

There are pre-built binaries for macos and linux attached to releases.
Just download and run them, or use a tool like [bin](https://github.com/marcosnils/bin) to do that for you.

## Configuration

The CLI stores configuration in `~/.config/outline-cli/config.yaml`. The configuration file is automatically created when you add your first server.

**Security Note:** The CLI requires the certificate SHA256 hash for each server to verify the server's identity. This prevents man-in-the-middle attacks by ensuring you're connecting to the correct server.

Example configuration:
```yaml
servers:
  my-server:
    name: my-server
    url: https://myserver.com/SecretPath
    certSha256: 34B3C8EB1C6EC9B5335556D7E8DC73A30152D27C66B054BAB8ACF5D11AE0C810
  production:
    name: production
    url: https://prod-server.com/AnotherSecretPath
    certSha256: 1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF
```

## Usage

### Server Management

#### List all servers
```bash
outline-cli servers list
```

#### Add a new server
```bash
outline-cli servers add <server-name> <server-url> --cert-sha256 <certificate-hash>
```

Example:
```bash
outline-cli servers add my-server https://myserver.com/SecretPath --cert-sha256 34B3C8EB1C6EC9B5335556D7E8DC73A30152D27C66B054BAB8ACF5D11AE0C810
```

#### Add a server from JSON
```bash
outline-cli servers add <server-name> --json '{"apiUrl": "https://server.com:port/path", "certSha256": "certificate-hash"}'
```

Example:
```bash
outline-cli servers add production --json '{"apiUrl": "https://vpn.drunkcoding.net:60000/b782eecb-bb9e-58be-614a-d5de1431d6b3", "certSha256": "34B3C8EB1C6EC9B5335556D7E8DC73A30152D27C66B054BAB8ACF5D11AE0C810"}'
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
outline-cli servers keys create <server-name> [--name <key-name>] [--method <encryption-method>] [--port <port>] [--data-limit <size>]
```

Example:
```bash
outline-cli servers keys create my-server --name "My Key" --method aes-192-gcm --port 12345
```

Example with data limit (1GB):
```bash
outline-cli servers keys create my-server --name "Limited Key" --data-limit 1GB
# supports human-readable sizes like `1GB`, `500MB`, `2TB`, `1.5GB`, etc.
```

#### Edit an access key
```bash
outline-cli servers keys edit <server-name> [--key-id <key-id> | --key-name <key-name>] [--new-name <new-name>] [--data-limit <size>] [--remove-limit]
```

Examples:
```bash
# Rename a key by ID
outline-cli servers keys edit my-server --key-id "1" --new-name "Updated Key"

# Change data limit by name
outline-cli servers keys edit my-server --key-name "My Key" --data-limit 2GB

# Remove data limit
outline-cli servers keys edit my-server --key-name "My Key" --remove-limit

# Rename and set new limit
outline-cli servers keys edit my-server --key-name "Old Name" --new-name "New Name" --data-limit 1.5GB
```

#### Delete an access key
```bash
outline-cli servers keys delete <server-name> --key-id <key-id>
```

Or delete by name:
```bash
outline-cli servers keys delete <server-name> --key-name <key-name>
```

**Note:** Key ID and Key Name are different:
- **Key ID**: A unique identifier assigned by the server (e.g., "1", "2", "abc123")
- **Key Name**: A human-readable name you assigned when creating the key (e.g., "My Key", "Production Key")

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

### Building

```bash
just build
```
or
```bash
nix build
```

### Running tests

```bash
just lint
just test
```

## Setup an Outline Shadowsocks VPN Server

1. Get a machine outside of firewall (VMs, VPS, etc.)
2. Install outline using [official script](https://raw.githubusercontent.com/Jigsaw-Code/outline-server/master/src/server_manager/install_scripts/install_server.sh) (read the script before running)
3. Add the server to `outline-cli` using json key from the script
4. Generate keys for your clients
5. ???
6. PROFIT

## License

This project is licensed under the MIT License.
The author is not affiliated with Jigsaw or Google, all respective trademarks are reserved.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

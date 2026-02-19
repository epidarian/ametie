# Ametie C2 Reverse Tunnel Service

Ametie is a cross-platform command and control (C2) service with reverse tunneling capabilities, designed for connectivity preservation across network boundaries. It provides secure, obfuscated communication between nodes and a central server.

## Features

- **Cross-Platform**: Supports Windows, Linux, and macOS
- **Obfuscated Communication**: API keys never transmitted, HMAC-based request signing
- **Multi-Strategy Phone Home**: Aggressive connection attempts with multiple fallback strategies
- **Command Execution**: Queue and execute commands on remote nodes
- **Reverse Tunnels**: Establish SSH tunnels between nodes
- **Unified Mailbox System**: Automatic command output capture, node-to-node messaging, and general notifications
- **Firewall Evasion**: Multiple transport protocols and obfuscation techniques
- **Auto-Cleanup**: Commands, tunnels, and messages auto-expire

## Architecture

### Components

1. **Go Client Service** (`ametie-client`): Background daemon that checks in with the server, executes commands, and manages tunnels
2. **Go CLI Tool** (`ametie`): Command-line interface for managing nodes, sending commands, and viewing status
3. **PHP-MySQL Server**: REST API backend with obscured endpoints for node management

### Security

- **API Key Obfuscation**: API keys are never transmitted. Requests use HMAC-SHA256 signatures
- **Request Encryption**: Request bodies are XOR-encrypted with derived keys
- **Header Rotation**: Custom headers rotate daily to avoid fingerprinting
- **Failure Obfuscation**: Returns 404 on authentication failures until 9th attempt from same IP
- **Timestamp Validation**: Prevents replay attacks with 5-minute window

## Installation

### Server Setup

1. **Database Setup**:
   ```bash
   mysql -u root -p < server/database/schema.sql
   ```

2. **Configure Database**:
   Edit `server/config.php` and update database credentials:
   ```php
   define('DB_HOST', 'localhost');
   define('DB_NAME', 'ametie');
   define('DB_USER', 'ametie_user');
   define('DB_PASS', 'your_password');
   ```

3. **Upload PHP Files**:
   Upload all files from `server/` to your web server (e.g., `domain.com/`)

4. **Set Permissions**:
   ```bash
   chmod 644 server/*.php
   ```

### Client Installation

#### Linux/macOS

```bash
sudo ./install/install.sh
```

The script will:
- Prompt for API key, server URL, and node name
- Build and install binaries
- Configure the service
- Install as systemd service (Linux) or launchd (macOS)

#### Windows

**PowerShell (Recommended)**:
```powershell
.\install\install.ps1
```

**Batch Script**:
```cmd
install\install.bat
```

**Manual Installation**:
1. Build binaries:
   ```bash
   go build -o ametie.exe ./cmd/ametie
   go build -o ametie-client.exe ./cmd/ametie-client
   ```

2. Install service:
   ```cmd
   sc create Ametie binPath= "C:\path\to\ametie-client.exe" start= auto
   sc start Ametie
   ```

## Usage

### Configuration

```bash
# Initial installation and configuration
ametie install --api-key KEY --server-url https://domain.com --node-name mynode

# Update configuration
ametie configure --secure-end-enable

# Configure server endpoints
ametie configure endpoint https://primary.example.com --priority 1
ametie configure endpoint https://backup.example.com --priority 2 --mirror
ametie configure endpoint https://new-cluster.com --new-cluster  # Clears all existing endpoints

# Endpoint flags:
#   --new-cluster    Delete all current server data (new cluster) - shows warning
#   --priority int   Endpoint priority (lower = higher priority, default: 1 for primary, 10 for mirror)
#   --mirror         Mark endpoint as mirror/backup
```

**Endpoint Management:**
- Multiple endpoints can be configured with different priorities
- Lower priority numbers = higher priority (1 is highest)
- Mirrors are backup endpoints used when primary is unavailable
- The system automatically warns if connecting to a different cluster domain
- Use `--new-cluster` to switch to a completely different cluster (clears all existing endpoints)

### Node Management

```bash
# List all registered nodes
ametie list nodes

# Show local node status
ametie status

# Rename local node
ametie rename new-name
```

### Command Execution

```bash
# Queue a command for execution on a node
ametie send-command pc_name "ls -la"

# Queue command from file
ametie send-command pc_name --file script.sh

# List pending commands
ametie commands list

# Cancel a command
ametie commands cancel <command-id>
```

### Tunnels

```bash
# Create reverse tunnel
ametie tunnel --foreign-port 50111 localhost:8080

# List active tunnels
ametie tunnel list

# Close a tunnel
ametie tunnel close <tunnel-id>

# Connect via SSH
ametie connect ssh pc_name
```

### Mailbox (Unified Messaging System)

The mailbox system handles command output, node-to-node messages, and general notifications in one unified interface.

```bash
# Send message to general mailbox (no --host)
ametie compose "message"

# Send message to specific node
ametie compose "message" --host hostname

# Check mailbox for new items (command output, messages, notifications)
ametie mailbox check

# Read specific mailbox entry
ametie mailbox read <message-id>

# List all mailbox entries and messages
ametie mailbox list [--node NODE] [--unread]

# Clear mailbox entries and messages
ametie mailbox clear [--node NODE] [--older-than DAYS] [--read]

# Export mailbox data
ametie mailbox export [--format json|csv]
```

### Service Management

```bash
# Start service
ametie service start

# Stop service
ametie service stop

# Restart service
ametie service restart

# View logs
ametie service logs [--tail N]
```

## Password Management

Passwords can be provided in several ways:

1. **Environment Variable**:
   ```bash
   export AMETIE_PASSWORD=your_password
   ```

2. **Session Cache**: Password is cached in memory after first use in a session

3. **Secure Endpoint Mode**: Use `--secure-end-enable` to cache password persistently in OS keychain

## API Endpoints

The server exposes the following obscured endpoints:

- `POST /sync.php` - Node heartbeat/check-in
- `GET /list.php` - List nodes (CLI)
- `POST /submit.php` - Queue command
- `GET /fetch.php` - Get pending commands
- `POST /request.php` - Register tunnel request
- `GET /status.php` - Get tunnel requests
- `POST /messages.php` - Write to mailbox/send message
- `GET /messages.php` - Read mailbox/messages
- `DELETE /messages.php` - Clear mailbox/messages

## Network Resilience

The client service implements multiple strategies for phone-home:

- **Multiple Endpoints**: Configure primary and mirror endpoints with priority-based failover
- **Primary**: HTTPS on standard ports (443, 8443)
- **Fallback**: HTTP on standard ports (80, 8080)
- **Advanced**: WebSocket, DNS tunneling, ICMP tunneling
- **Proxy Support**: Auto-detection of HTTP CONNECT and SOCKS proxies
- **NAT Traversal**: UPnP, STUN-like techniques
- **Protocol Obfuscation**: Browser-like headers and request patterns

**Endpoint Failover:**
- The service automatically tries endpoints in priority order
- Mirrors are used when primary endpoints fail
- Cluster detection prevents accidental connection to wrong servers
- Endpoints can be added/removed dynamically via `configure endpoint`

## Database Schema

- `nodes`: Registered nodes with hostname, custom name, and status
- `commands`: Queued commands with execution status
- `tunnels`: Active tunnel requests and status
- `mailbox`: Command output, notifications, and general messages (unified)
- `messages`: Node-to-node direct messages (also accessible via mailbox)
- `failed_attempts`: Authentication failure tracking
- `api_keys`: API key management

## Development

### Building

```bash
# Build CLI tool
go build -o bin/ametie ./cmd/ametie

# Build client service
go build -o bin/ametie-client ./cmd/ametie-client
```

### Project Structure

```
ametie/
├── cmd/
│   ├── ametie/           # CLI tool
│   └── ametie-client/    # Service daemon
├── internal/
│   ├── client/           # Client service logic
│   ├── cli/              # CLI command handlers
│   ├── ssh/              # SSH tunnel management
│   ├── config/           # Configuration management
│   ├── mailbox/          # Mailbox operations
│   ├── heartbeat/        # Multi-strategy heartbeat
│   ├── auth/             # Password/session management
│   └── network/          # Network traversal and firewall evasion
├── server/               # PHP API endpoints
├── install/              # Installation scripts
└── README.md
```

## Security Considerations

- API keys are stored encrypted in OS keychain/credential store
- All requests are signed with HMAC-SHA256
- Request bodies are encrypted with XOR using derived keys
- Failed authentication attempts return 404 to obscure detection
- Timestamp validation prevents replay attacks
- Header rotation reduces traffic fingerprinting

## Troubleshooting

### Service Not Starting

- Check logs: `ametie service logs`
- Verify configuration: `ametie status`
- Ensure API key and server URL are correct

### Connection Issues

- Verify server URL is accessible
- Check firewall rules
- Review network transport logs
- Try different ports or protocols

### Command Execution Fails

- Check mailbox for error output: `ametie mailbox check`
- Verify node is online: `ametie list nodes`
- Review command syntax and permissions

## License

[Specify your license here]

## Contributing

[Contributing guidelines]

## Support

[Support information]


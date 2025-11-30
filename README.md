# Light Shadowsocks Client

A lightweight shadowsocks client written in Go that provides local HTTP/HTTPS and SOCKS5 proxy servers. Forward all your traffic through a shadowsocks server with ease.

## Features

- **Shadowsocks Client**: Connect to any shadowsocks server with AEAD cipher support
- **Unified Proxy Mode**: Single port for HTTP/HTTPS and SOCKS5 (like Clash)
- **Separate Proxy Mode**: Dedicated ports for HTTP and SOCKS5
- **Simple-obfs Plugin**: HTTP and TLS obfuscation support
- **Command-line Parameters**: Run without config files - perfect for automation
- **Config Converters**: Import from ss-local and Clash configurations
- **Statistics Monitoring**: Track connections and bandwidth usage
- **Graceful Shutdown**: Proper cleanup on exit signals
- **Flexible Configuration**: YAML/JSON config files, environment variables, or CLI params

## Installation

### From Source

```bash
git clone https://github.com/xrdavies/light-ss.git
cd light-ss
make build
```

The binary will be available at `./bin/light-ss`.

### Manual Build

```bash
go build -o light-ss ./cmd/light-ss
```

## Quick Start

### Using Configuration File

1. Create a configuration file:

```bash
cp config.yaml.example config.yaml
```

2. Edit `config.yaml` with your shadowsocks server details:

```yaml
shadowsocks:
  server: "your-server.com:8388"
  password: "your-password"
  cipher: "AEAD_CHACHA20_POLY1305"

# Unified mode - single port for both HTTP and SOCKS5
proxies: "127.0.0.1:1080"
```

3. Start the client:

```bash
./light-ss start -c config.yaml
```

4. Configure your applications to use the proxy at `127.0.0.1:1080`

### Using Command-line Parameters (No Config File)

```bash
./light-ss start \
  --server your-server.com \
  --port 8388 \
  --password your-password \
  --method aes-128-gcm \
  --proxies 127.0.0.1:1080
```

### With Simple-obfs Plugin

```bash
./light-ss start \
  --server your-server.com \
  --port 8388 \
  --password your-password \
  --method aes-128-gcm \
  --plugin simple-obfs \
  --plugin-obfs http \
  --plugin-host www.bing.com \
  --proxies 127.0.0.1:1080
```

## Usage

### Start the Client

```bash
# With config file
./light-ss start -c config.yaml

# With CLI parameters only
./light-ss start -s server.com -p 8388 --password pass -m aes-128-gcm

# Override config file values
./light-ss start -c config.yaml --log-level debug --proxies 0.0.0.0:1080

# View all available flags
./light-ss start --help
```

### Convert Existing Configurations

```bash
# Convert from ss-local format
./light-ss convert --from ss-local --input ss-local.json --output config.json

# Convert from Clash format
./light-ss convert --from clash --input clash.yaml --output config.yaml

# Print to stdout (JSON format)
./light-ss convert --from ss-local --input ss-local.json
```

### Testing the Proxies

```bash
# Unified mode (single port for both protocols)
curl -x socks5://127.0.0.1:1080 https://www.google.com
curl -x http://127.0.0.1:1080 https://www.google.com

# Separate mode
curl -x socks5://127.0.0.1:1080 https://www.google.com
curl -x http://127.0.0.1:8080 https://www.google.com
```

### Browser Configuration

#### Firefox
1. Settings → Network Settings → Manual proxy configuration
2. SOCKS Host: `127.0.0.1`, Port: `1080`
3. Check "SOCKS v5" and "Proxy DNS when using SOCKS v5"

**Note:** With unified mode, both HTTP and SOCKS5 work on the same port!

#### Chrome/Chromium
```bash
# Unified mode - use either SOCKS5 or HTTP on same port
chromium --proxy-server="socks5://127.0.0.1:1080"
chromium --proxy-server="http://127.0.0.1:1080"

# Separate mode
chromium --proxy-server="socks5://127.0.0.1:1080"
chromium --proxy-server="http://127.0.0.1:8080"
```

## Configuration

### Configuration File

The configuration file supports both YAML and JSON formats. See `config.yaml.example` or `config.json.example` for complete examples.

#### Shadowsocks Settings

```yaml
shadowsocks:
  server: "example.com:8388"          # Server address and port
  password: "your-password"           # Server password
  cipher: "AEAD_CHACHA20_POLY1305"   # Encryption cipher
  timeout: 300                        # Connection timeout (seconds)

  # Optional: Simple-obfs plugin
  plugin: "simple-obfs"
  plugin_opts:
    obfs: "http"                      # or "tls"
    obfs-host: "www.bing.com"
```

**Supported Ciphers:**
- `AEAD_CHACHA20_POLY1305` (recommended)
- `AEAD_AES_256_GCM`
- `AEAD_AES_128_GCM`
- `aes-128-gcm`, `aes-256-gcm`, `chacha20-poly1305` (auto-normalized)

#### Proxy Settings

**Unified Mode (Recommended):** Single port for both HTTP/HTTPS and SOCKS5

```yaml
# YAML format
proxies: "127.0.0.1:1080"
```

```json
// JSON format
{
  "proxies": "127.0.0.1:1080"
}
```

**Separate Mode:** Dedicated ports for each protocol

```yaml
# YAML format
proxies:
  http: "127.0.0.1:8080"
  socks5: "127.0.0.1:1080"
```

```json
// JSON format
{
  "proxies": {
    "http": "127.0.0.1:8080",
    "socks5": "127.0.0.1:1080"
  }
}
```

**SOCKS5 with Authentication:**

```yaml
proxies:
  socks5: "user:pass@127.0.0.1:1080"
```

#### Statistics

```yaml
stats:
  enabled: true     # Enable stats collection
  interval: 60      # Report interval in seconds
```

When enabled, statistics will be logged periodically showing:
- Total and active connections
- HTTP and SOCKS5 connection counts
- Bytes sent and received
- Uptime

#### Logging

```yaml
logging:
  level: "info"     # debug, info, warn, error
  format: "text"    # json, text
```

### Command-line Parameters

All configuration can be specified via command-line flags:

```bash
./light-ss start [flags]
```

**Shadowsocks Flags:**
- `-s, --server string` - Shadowsocks server address
- `-p, --port int` - Shadowsocks server port
- `--password string` - Shadowsocks password
- `-m, --method string` - Encryption method (aes-128-gcm, aes-256-gcm, chacha20-poly1305)
- `--timeout int` - Connection timeout in seconds

**Plugin Flags:**
- `--plugin string` - Plugin name (e.g., simple-obfs)
- `--plugin-obfs string` - Obfuscation mode: http or tls
- `--plugin-host string` - Obfuscation host header

**Proxy Flags:**
- `--proxies string` - Unified proxy listen address (e.g., 127.0.0.1:1080)
- `--http-proxy string` - HTTP/HTTPS proxy listen address
- `--socks5-proxy string` - SOCKS5 proxy listen address (supports user:pass@host:port)

**Other Flags:**
- `-c, --config string` - Path to configuration file (optional)
- `--log-level string` - Log level (debug, info, warn, error)

**Examples:**

```bash
# Minimal setup
./light-ss start -s server.com -p 8388 --password pass -m aes-128-gcm

# With plugin
./light-ss start \
  -s server.com -p 8388 --password pass -m aes-128-gcm \
  --plugin simple-obfs --plugin-obfs http --plugin-host www.bing.com

# Separate proxy ports
./light-ss start -s server.com -p 8388 --password pass -m aes-128-gcm \
  --http-proxy 127.0.0.1:8080 --socks5-proxy 127.0.0.1:1080

# Override config file
./light-ss start -c config.yaml --log-level debug --proxies 0.0.0.0:1080
```

### Environment Variables

You can override configuration values using environment variables:

```bash
export LIGHT_SS_SERVER="example.com:8388"
export LIGHT_SS_PASSWORD="your-password"
export LIGHT_SS_CIPHER="AEAD_CHACHA20_POLY1305"
export LIGHT_SS_HTTP_LISTEN="127.0.0.1:8080"
export LIGHT_SS_SOCKS5_LISTEN="127.0.0.1:1080"
export LIGHT_SS_LOG_LEVEL="debug"
```

**Priority:** CLI flags > Environment variables > Config file > Defaults

## Architecture

### Unified Mode (Default)

Single port handles both HTTP/HTTPS and SOCKS5 through protocol detection:

```
Client Apps → Unified Proxy:1080 → Stats → Shadowsocks+Plugin → Server → Internet
              (HTTP or SOCKS5)
```

### Separate Mode (Optional)

Dedicated ports for each protocol:

```
Client Apps → HTTP:8080 ────┐
              SOCKS5:1080 ───┴→ Stats → Shadowsocks → Server → Internet
```

## Config Converters

Light-ss can import configurations from other shadowsocks clients:

### From ss-local (shadowsocks-libev)

```bash
# Convert and save
./light-ss convert --from ss-local --input ss-local.json --output config.json

# Print to stdout
./light-ss convert --from ss-local --input ss-local.json
```

**ss-local format** (JSON):
```json
{
  "server": "example.com",
  "server_port": 8388,
  "password": "password",
  "method": "aes-128-gcm",
  "plugin": "obfs-local",
  "plugin_opts": "obfs=http;obfs-host=www.bing.com"
}
```

### From Clash

```bash
# Convert to YAML
./light-ss convert --from clash --input clash.yaml --output config.yaml

# Convert to JSON
./light-ss convert --from clash --input clash.yaml --output config.json
```

**Clash format** (YAML):
```yaml
proxies:
  - name: "ss-server"
    type: ss
    server: example.com
    port: 8388
    cipher: aes-128-gcm
    password: "password"
    plugin: obfs
    plugin-opts:
      mode: http
      host: www.bing.com
```

## Security Considerations

1. **Cipher Selection**: Always use AEAD ciphers (ChaCha20-Poly1305 or AES-GCM)
2. **Plugin Obfuscation**: Use simple-obfs to disguise traffic as HTTP/TLS
3. **Local Only**: Proxies bind to `127.0.0.1` by default - don't expose them publicly
4. **Config File**: Protect your config file permissions: `chmod 600 config.yaml`
5. **No Logging**: Sensitive data like passwords are never logged
6. **HTTPS**: The client doesn't inspect HTTPS traffic - it only tunnels it

## Troubleshooting

### Enable Debug Logging

```bash
./light-ss start -c config.yaml --log-level debug
```

### Connection Issues

- Verify shadowsocks server is running and accessible
- Check cipher matches server configuration
- Ensure password is correct
- Test server connectivity: `telnet your-server.com 8388`

### Proxy Not Working

- Verify the proxy server started (check logs)
- Test local connectivity: `curl http://127.0.0.1:8080`
- Check firewall rules
- Ensure no other service is using the same port

### Statistics Not Showing

- Enable stats in config: `stats.enabled: true`
- Check stats interval setting
- View logs for statistics output

## Building

### Build for Current Platform

```bash
make build
```

### Build for All Platforms

```bash
make build-all
```

This creates binaries for:
- Linux (amd64)
- macOS (amd64, arm64)
- Windows (amd64)

## Dependencies

- [go-shadowsocks2](https://github.com/shadowsocks/go-shadowsocks2) - Shadowsocks client
- [go-socks5](https://github.com/armon/go-socks5) - SOCKS5 server
- [goproxy](https://github.com/elazarl/goproxy) - HTTP/HTTPS proxy
- [cobra](https://github.com/spf13/cobra) - CLI framework
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parser

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## Author

xrdavies

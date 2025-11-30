# Light Shadowsocks Client

A lightweight shadowsocks client written in Go that provides local HTTP/HTTPS and SOCKS5 proxy servers. Forward all your traffic through a shadowsocks server with ease.

## Features

- **Shadowsocks Client**: Connect to any shadowsocks server with AEAD cipher support
- **HTTP/HTTPS Proxy**: Local HTTP proxy with HTTPS CONNECT support
- **SOCKS5 Proxy**: Standard SOCKS5 proxy server
- **Statistics Monitoring**: Track connections and bandwidth usage
- **Debug Logging**: Verbose logging mode for troubleshooting
- **Graceful Shutdown**: Proper cleanup on exit signals
- **Easy Configuration**: YAML-based configuration with environment variable overrides

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

1. Create a configuration file:

```bash
cp configs/config.yaml.example config.yaml
```

2. Edit `config.yaml` with your shadowsocks server details:

```yaml
shadowsocks:
  server: "your-server.com:8388"
  password: "your-password"
  cipher: "AEAD_CHACHA20_POLY1305"
```

3. Start the client:

```bash
./light-ss start -c config.yaml
```

4. Configure your applications to use the local proxies:
   - HTTP/HTTPS: `http://127.0.0.1:8080`
   - SOCKS5: `socks5://127.0.0.1:1080`

## Usage

### Start the Client

```bash
# Start with config file
./light-ss start -c config.yaml

# Start with debug logging
./light-ss start -c config.yaml --log-level debug

# View help
./light-ss --help
./light-ss start --help
```

### Testing the Proxies

```bash
# Test SOCKS5 proxy
curl -x socks5://127.0.0.1:1080 https://www.google.com

# Test HTTP proxy
curl -x http://127.0.0.1:8080 https://www.google.com

# Test HTTPS through HTTP proxy
curl -x http://127.0.0.1:8080 https://api.github.com
```

### Browser Configuration

#### Firefox
1. Settings → Network Settings → Manual proxy configuration
2. SOCKS Host: `127.0.0.1`, Port: `1080`
3. Check "SOCKS v5" and "Proxy DNS when using SOCKS v5"

#### Chrome/Chromium
```bash
# Linux/macOS
chromium --proxy-server="socks5://127.0.0.1:1080"

# Or use HTTP proxy
chromium --proxy-server="http://127.0.0.1:8080"
```

## Configuration

### Configuration File

The configuration file uses YAML format. See `configs/config.yaml.example` for a complete example.

#### Shadowsocks Settings

```yaml
shadowsocks:
  server: "example.com:8388"          # Server address and port
  password: "your-password"           # Server password
  cipher: "AEAD_CHACHA20_POLY1305"   # Encryption cipher
  timeout: 300                        # Connection timeout (seconds)
```

**Supported Ciphers:**
- `AEAD_CHACHA20_POLY1305` (recommended)
- `AEAD_AES_256_GCM`
- `AEAD_AES_128_GCM`

#### Proxy Settings

```yaml
proxies:
  http:
    enabled: true
    listen: "127.0.0.1:8080"
  socks5:
    enabled: true
    listen: "127.0.0.1:1080"
    # Optional authentication
    # auth:
    #   username: "user"
    #   password: "pass"
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

### Command-line Flags

```bash
./light-ss start -c config.yaml --log-level debug
```

**Priority:** CLI flags > Environment variables > Config file > Defaults

## Architecture

```
┌─────────────────────────────────────────────────────┐
│              Client Applications                    │
│         (Browser, curl, wget, etc.)                 │
└──────────────┬──────────────────────────────────────┘
               │
               ├─────────────┬────────────────┐
               │             │                │
       ┌───────▼──────┐ ┌───▼────────┐       │
       │ HTTP Proxy   │ │ SOCKS5     │       │
       │ :8080        │ │ Proxy      │       │
       │              │ │ :1080      │       │
       └───────┬──────┘ └───┬────────┘       │
               │            │                │
               └────────┬───┘                │
                        │                    │
               ┌────────▼────────┐           │
               │ Stats Collector │◄──────────┘
               │ (Optional)      │
               └────────┬────────┘
                        │
               ┌────────▼────────────┐
               │ Shadowsocks Client  │
               │ Dialer              │
               └────────┬────────────┘
                        │
                        │ Encrypted
                        │
               ┌────────▼────────────┐
               │ Shadowsocks Server  │
               │ (Remote)            │
               └────────┬────────────┘
                        │
               ┌────────▼────────────┐
               │ Internet            │
               └─────────────────────┘
```

## Security Considerations

1. **Cipher Selection**: Always use AEAD ciphers (ChaCha20-Poly1305 or AES-GCM)
2. **Local Only**: Proxies bind to `127.0.0.1` by default - don't expose them publicly
3. **Config File**: Protect your config file permissions: `chmod 600 config.yaml`
4. **No Logging**: Sensitive data like passwords are never logged
5. **HTTPS**: The client doesn't inspect HTTPS traffic - it only tunnels it

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

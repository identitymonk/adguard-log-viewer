# AdGuard Log Viewer

A lightweight web interface for viewing AdGuardHome DNS query logs. Designed for resource-constrained routers like the GL-iNet GL-MT300 (MIPS CPU, 128MB RAM).

Single static binary, no dependencies, server-side rendered HTML.

## Features

- Browse DNS query logs in a sortable table (newest first)
- Filter by client IP, hostname, time range, and block status
- Filters combine with AND logic and are preserved in the URL for bookmarking
- Blocked queries highlighted in red
- Paginated results (50 per page)
- Streams the log file line-by-line to minimize memory usage

## Build

Requires Go 1.25+.

```sh
# Native build
make build

# Cross-compile for MIPS/OpenWrt (GL-MT300, etc.)
make build-mips

# Run tests
make test
```

## Configuration

Copy and edit the example config:

```sh
cp config.example.txt config.txt
```

Config format (`config.txt`):

```
# Path to AdGuardHome query log (NDJSON format)
log_file = /var/log/adguardhome/querylog.json

# HTTP port for the web interface
http_port = 8080
```

## Usage

```sh
# Run with default config path (config.txt)
./adguard-log-viewer

# Run with custom config path
./adguard-log-viewer /etc/adguard-log-viewer/config.txt
```

Then open `http://<router-ip>:8080` in your browser.

## Deploy to OpenWrt Router

1. Cross-compile:
   ```sh
   make build-mips
   ```

2. Copy files to the router:
   ```sh
   scp adguard-log-viewer-mips root@<router-ip>:/usr/bin/adguard-log-viewer
   scp template.html root@<router-ip>:/etc/adguard-log-viewer/template.html
   scp config.example.txt root@<router-ip>:/etc/adguard-log-viewer/config.txt
   ```

3. Edit the config on the router:
   ```sh
   ssh root@<router-ip>
   vi /etc/adguard-log-viewer/config.txt
   ```

4. Install the init script for auto-start on boot:
   ```sh
   scp scripts/adguard-log-viewer.init root@<router-ip>:/etc/init.d/adguard-log-viewer
   ssh root@<router-ip> "chmod +x /etc/init.d/adguard-log-viewer && /etc/init.d/adguard-log-viewer enable"
   ```

5. Start the service:
   ```sh
   ssh root@<router-ip> "/etc/init.d/adguard-log-viewer start"
   ```

## URL Query Parameters

| Parameter  | Description                                  |
|------------|----------------------------------------------|
| `ip`       | Filter by client IP (substring match)        |
| `hostname` | Filter by hostname (case-insensitive substr) |
| `start`    | Start time (datetime-local format)           |
| `end`      | End time (datetime-local format)             |
| `status`   | `blocked`, `allowed`, or empty for all       |
| `page`     | Page number (1-based)                        |

Example: `http://<router-ip>:8080/?ip=192.168&status=blocked&page=1`

## License

MIT

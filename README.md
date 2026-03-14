# Parameter Store

A lightweight, crash-safe, append-only parameter store written in Go.

## Features

- **Append-only storage**: Full audit trail in JSONL format with operation type (insert/update/delete) and client IP
- **In-memory index**: O(1) lookups for the latest value of any key
- **Atomic batch updates**: Multiple parameters updated atomically with file locking + fsync
- **Crash safety**: Data is either fully persisted or not at all
- **Password masking**: Password-type values masked by default, reveal on demand
- **Web UI**: Batch editing with change preview before saving

## Quick Start

```bash
# Build (with stripped symbols for smaller binary)
go build -ldflags="-s -w" -o parameter-store .

# Run
./parameter-store

# Custom port and data file
./parameter-store -port 3000 -data /path/to/store.jsonl
```

Open http://localhost:8847

## API

### POST /api/update

Batch update parameters. Each record is tagged with operation type and client IP.

```bash
# Insert/Update
curl -X POST http://localhost:8847/api/update \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"key": "db_host", "value": "localhost", "type": "text"},
      {"key": "db_password", "value": "secret", "type": "password"}
    ]
  }'

# Delete
curl -X POST http://localhost:8847/api/update \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"key": "db_host", "is_delete": true}
    ]
  }'
```

### GET /api/list

List all parameters. Passwords masked by default.

```bash
curl http://localhost:8847/api/list
curl http://localhost:8847/api/list?unmask=true
```

### GET /api/get

Get single parameter (unmasked).

```bash
curl http://localhost:8847/api/get?key=db_password
```

## Data Format

Each line in the JSONL file is a complete record:

```json
{"key":"host","value":"localhost","type":"text","operation":"insert","timestamp":1720000000000,"ip":"192.168.1.1"}
{"key":"host","value":"prod.example.com","type":"text","operation":"update","timestamp":1720000060000,"ip":"192.168.1.1"}
{"key":"secret","value":"","type":"text","operation":"delete","timestamp":1720000120000,"ip":"192.168.1.1"}
```

Fields:
- `key`: Parameter name
- `value`: Parameter value (empty for deletes)
- `type`: `text` or `password`
- `operation`: `insert`, `update`, or `delete`
- `timestamp`: Unix milliseconds
- `ip`: Client IP address

## Crash Safety

1. Acquire mutex lock (prevents concurrent writes)
2. Append records to file
3. Call fsync (forces data to disk)
4. Update in-memory index (only after fsync succeeds)
5. Release lock

If crash occurs before fsync: data not persisted, index unchanged.
If crash occurs after fsync: data on disk, index rebuilt on restart.

## Project Structure

```
.
├── main.go                    # Entry point
├── internal/
│   ├── models/models.go       # Data structures
│   ├── store/store.go         # Core store logic
│   └── api/handlers.go        # HTTP handlers
├── web/
│   └── index.html             # Web UI
└── examples/
    └── demo.sh                # Demo script
```

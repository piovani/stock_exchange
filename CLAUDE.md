# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make proto   # regenerate Go code from .proto files (requires protoc in PATH)
make build   # compile server binary → bin/server
make run     # run server locally (default port 50051, override with PORT=)
make test    # go test ./...
make lint    # go vet ./...
```

Run a single test package:
```bash
go test ./internal/domain/stock/...
```

## Architecture

gRPC API for a stock exchange backed by Yahoo Finance (no API key required).

```
proto/stock/stock.proto              # source of truth for all service/message definitions
pb/stock/                            # generated — never edit by hand (run make proto)

cmd/server/main.go                   # wires dependencies and starts the gRPC server

internal/
  domain/stock/
    entity.go                        # Quote, SearchResult structs
    repository.go                    # Repository interface (port)
    service.go                       # domain service + ErrNotFound sentinel
  infra/yahoofinance/
    client.go                        # shared HTTP client with User-Agent
    dto.go                           # JSON response structs for Yahoo Finance
    repository.go                    # implements domain Repository via Yahoo Finance API
  grpc/stock/
    handler.go                       # maps gRPC requests → domain service → proto responses
```

**Dependency flow:** `grpc/handler → domain/service → domain/repository ← infra/yahoofinance`

**Adding a new service:**
1. Add RPC + messages to `proto/stock/stock.proto` (or create a new proto file)
2. `make proto`
3. Add method to `internal/domain/stock/service.go` + `repository.go`
4. Implement in `internal/infra/yahoofinance/repository.go`
5. Add handler method in `internal/grpc/stock/handler.go`
6. Register in `cmd/server/main.go`

## gRPC reflection

The server exposes gRPC reflection — use `grpcurl` without a proto file:

```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext -d '{"symbol":"AAPL"}' localhost:50051 stock.StockService/GetQuote
grpcurl -plaintext -d '{"symbols":["AAPL","MSFT"]}' localhost:50051 stock.StockService/ListQuotes
grpcurl -plaintext -d '{"query":"apple"}' localhost:50051 stock.StockService/SearchStocks
```

## Yahoo Finance integration

- **GetQuote / ListQuotes**: `GET /v8/finance/chart/{symbol}` — real-time price, change, % change
- **SearchStocks**: `GET /v1/finance/search?q={query}` — symbol search with exchange and type
- `ListQuotes` fetches all symbols concurrently (goroutine per symbol); individual failures are silently skipped
- No API key needed; requests use a browser `User-Agent` header

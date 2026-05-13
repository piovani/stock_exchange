# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make proto      # regenerate Go code from .proto files (requires protoc in PATH)
make build      # compile server binary → bin/server
make run        # run server locally (default port 50051, override with PORT=)
make test       # go test ./...
make lint       # go vet ./...
make db-up      # start PostgreSQL via Docker Compose and wait until ready
make db-down    # stop Docker Compose services
make collector  # run the history collector (pass args with ARGS=)
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
cmd/collector/main.go                # CLI: backfills + watches OHLCV history into Postgres

internal/
  domain/stock/
    entity.go                        # Quote, SearchResult structs
    history.go                       # HistoricalQuote, HistoricalRepository, HistoricalClient interfaces
    repository.go                    # Repository interface (port)
    service.go                       # domain service + ErrNotFound sentinel
  infra/postgres/
    db.go                            # pgxpool.Open helper
    repository.go                    # implements HistoricalRepository (upsert + query)
  provider/yahoofinance/
    client.go                        # shared HTTP client with User-Agent
    dto.go                           # JSON response structs for Yahoo Finance
    historical.go                    # GetHistoricalQuotes — daily OHLCV via chart endpoint
    repository.go                    # implements domain Repository via Yahoo Finance API
  grpc/stock/
    handler.go                       # maps gRPC requests → domain service → proto responses
```

**Dependency flow:** `grpc/handler → domain/service → domain/repository ← provider/yahoofinance`

**Historical data flow:** `cmd/collector → domain/stock.HistoricalClient (provider/yahoofinance) → domain/stock.HistoricalRepository (infra/postgres)`

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
- **GetHistoricalQuotes**: `GET /v8/finance/chart/{symbol}?interval=1d&period1=…&period2=…` — daily OHLCV for a date range
- `ListQuotes` fetches all symbols concurrently (goroutine per symbol); individual failures are silently skipped
- No API key needed; requests use a browser `User-Agent` header

## Historical data collector

`cmd/collector` backfills the last 12 months of OHLCV data into PostgreSQL and optionally watches for daily updates.

```bash
# one-shot backfill (DATABASE_URL must be set)
make collector

# custom symbols
make collector ARGS="-symbols AAPL,MSFT"

# watch mode: re-collects daily at 23:00 UTC
make collector ARGS="-watch"
```

Requires a running PostgreSQL instance (`make db-up`) with `market_metadata` and `stock_quotes_history` tables.

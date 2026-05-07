package stock

import "context"

type Repository interface {
	GetQuote(ctx context.Context, symbol string) (Quote, error)
	// GetQuotes fetches multiple symbols concurrently; skips failures silently.
	GetQuotes(ctx context.Context, symbols []string) ([]Quote, error)
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

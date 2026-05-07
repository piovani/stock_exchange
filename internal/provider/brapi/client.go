package brapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/piovani/stock_exchange/internal/domain/stock"
)

const (
	availableURL = "https://brapi.dev/api/available"
	cacheTTL     = 24 * time.Hour
)

type Client struct {
	http     *http.Client
	mu       sync.RWMutex
	cache    []stock.Symbol
	cachedAt time.Time
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) ListSymbols(ctx context.Context) ([]stock.Symbol, error) {
	c.mu.RLock()
	if len(c.cache) > 0 && time.Since(c.cachedAt) < cacheTTL {
		out := make([]stock.Symbol, len(c.cache))
		copy(out, c.cache)
		c.mu.RUnlock()
		return out, nil
	}
	c.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, availableURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Stocks []string `json:"stocks"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	symbols := make([]stock.Symbol, len(data.Stocks))
	for i, ticker := range data.Stocks {
		symbols[i] = stock.Symbol{Ticker: ticker, Exchange: "B3", Type: "EQUITY"}
	}

	c.mu.Lock()
	c.cache = symbols
	c.cachedAt = time.Now()
	c.mu.Unlock()

	return symbols, nil
}

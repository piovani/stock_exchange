package nasdaq

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/piovani/stock_exchange/internal/domain/stock"
)

const (
	nasdaqListedURL = "https://www.nasdaqtrader.com/dynamic/SymDir/nasdaqlisted.txt"
	otherListedURL  = "https://www.nasdaqtrader.com/dynamic/SymDir/otherlisted.txt"
	cacheTTL        = 24 * time.Hour
)

// nasdaqlisted.txt: Symbol|Security Name|Market Category|Test Issue|Financial Status|Round Lot Size|ETF|NextShares
// otherlisted.txt:  ACT Symbol|Security Name|Exchange|CQS Symbol|ETF|Round Lot Size|Test Issue|NASDAQ Symbol

var otherExchanges = map[string]string{
	"A": "NYSE MKT",
	"N": "NYSE",
	"P": "NYSE ARCA",
	"Z": "BATS",
	"V": "IEX",
}

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

	type result struct {
		syms []stock.Symbol
		err  error
	}
	ch := make(chan result, 2)

	go func() {
		s, err := c.fetch(ctx, nasdaqListedURL, parseNasdaq)
		ch <- result{s, err}
	}()
	go func() {
		s, err := c.fetch(ctx, otherListedURL, parseOther)
		ch <- result{s, err}
	}()

	var all []stock.Symbol
	for range 2 {
		r := <-ch
		if r.err != nil {
			log.Printf("nasdaq client: %v", r.err)
			continue
		}
		all = append(all, r.syms...)
	}

	c.mu.Lock()
	c.cache = all
	c.cachedAt = time.Now()
	c.mu.Unlock()

	return all, nil
}

func (c *Client) fetch(ctx context.Context, url string, parse func(string) (stock.Symbol, bool)) ([]stock.Symbol, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var symbols []stock.Symbol
	scanner := bufio.NewScanner(resp.Body)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			continue
		}
		if strings.HasPrefix(line, "File Creation Time") {
			continue
		}
		if sym, ok := parse(line); ok {
			symbols = append(symbols, sym)
		}
	}
	return symbols, scanner.Err()
}

func parseNasdaq(line string) (stock.Symbol, bool) {
	parts := strings.Split(line, "|")
	if len(parts) < 8 || parts[3] == "Y" {
		return stock.Symbol{}, false
	}
	typ := "EQUITY"
	if parts[6] == "Y" {
		typ = "ETF"
	}
	return stock.Symbol{Ticker: parts[0], Name: parts[1], Exchange: "NASDAQ", Type: typ}, true
}

func parseOther(line string) (stock.Symbol, bool) {
	parts := strings.Split(line, "|")
	if len(parts) < 8 || parts[6] == "Y" {
		return stock.Symbol{}, false
	}
	exchange := otherExchanges[parts[2]]
	if exchange == "" {
		exchange = parts[2]
	}
	typ := "EQUITY"
	if parts[4] == "Y" {
		typ = "ETF"
	}
	return stock.Symbol{Ticker: parts[0], Name: parts[1], Exchange: exchange, Type: typ}, true
}

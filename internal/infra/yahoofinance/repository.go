package yahoofinance

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/piovani/stock_exchange/internal/domain/stock"
)

type Repository struct {
	client *Client
}

func NewRepository(client *Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) GetQuote(ctx context.Context, symbol string) (stock.Quote, error) {
	return r.fetchQuote(ctx, symbol)
}

func (r *Repository) GetQuotes(ctx context.Context, symbols []string) ([]stock.Quote, error) {
	type result struct {
		q   stock.Quote
		err error
	}

	ch := make([]result, len(symbols))
	var wg sync.WaitGroup

	for i, sym := range symbols {
		wg.Add(1)
		go func(idx int, s string) {
			defer wg.Done()
			q, err := r.fetchQuote(ctx, s)
			ch[idx] = result{q, err}
		}(i, sym)
	}
	wg.Wait()

	quotes := make([]stock.Quote, 0, len(symbols))
	for i, res := range ch {
		if res.err != nil {
			log.Printf("ListQuotes: skipping %q: %v", symbols[i], res.err)
			continue
		}
		quotes = append(quotes, res.q)
	}
	return quotes, nil
}

func (r *Repository) Search(ctx context.Context, query string) ([]stock.SearchResult, error) {
	endpoint := "https://query1.finance.yahoo.com/v1/finance/search?" + url.Values{
		"q":           {query},
		"lang":        {"en-US"},
		"region":      {"US"},
		"quotesCount": {"10"},
		"newsCount":   {"0"},
	}.Encode()

	var resp searchResponse
	if err := r.client.get(ctx, endpoint, &resp); err != nil {
		return nil, err
	}

	results := make([]stock.SearchResult, 0, len(resp.Quotes))
	for _, q := range resp.Quotes {
		name := normalizeSpace(q.ShortName)
		if name == "" {
			name = normalizeSpace(q.LongName)
		}
		results = append(results, stock.SearchResult{
			Symbol:    q.Symbol,
			ShortName: name,
			Exchange:  q.Exchange,
			Type:      q.QuoteType,
		})
	}
	return results, nil
}

func (r *Repository) fetchQuote(ctx context.Context, symbol string) (stock.Quote, error) {
	endpoint := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d",
		url.PathEscape(symbol),
	)

	var resp chartResponse
	if err := r.client.get(ctx, endpoint, &resp); err != nil {
		return stock.Quote{}, err
	}
	if resp.Chart.Error != nil {
		return stock.Quote{}, fmt.Errorf("yahoo finance: %s", resp.Chart.Error.Description)
	}
	if len(resp.Chart.Result) == 0 {
		return stock.Quote{}, fmt.Errorf("%w: %s", stock.ErrNotFound, symbol)
	}

	meta := resp.Chart.Result[0].Meta
	change := meta.RegularMarketPrice - meta.ChartPreviousClose
	var changePct float64
	if meta.ChartPreviousClose != 0 {
		changePct = change / meta.ChartPreviousClose * 100
	}

	return stock.Quote{
		Symbol:        meta.Symbol,
		ShortName:     normalizeSpace(meta.ShortName),
		Price:         meta.RegularMarketPrice,
		Change:        change,
		ChangePercent: changePct,
		Timestamp:     meta.RegularMarketTime,
	}, nil
}

// normalizeSpace collapses internal runs of whitespace into a single space.
func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

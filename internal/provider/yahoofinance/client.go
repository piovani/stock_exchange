package yahoofinance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/piovani/stock_exchange/internal/domain/stock"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

type Client struct {
	http *http.Client
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) GetQuote(ctx context.Context, symbol string) (stock.Quote, error) {
	return c.fetchQuote(ctx, symbol)
}

func (c *Client) GetQuotes(ctx context.Context, symbols []string) ([]stock.Quote, error) {
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
			q, err := c.fetchQuote(ctx, s)
			ch[idx] = result{q, err}
		}(i, sym)
	}
	wg.Wait()

	quotes := make([]stock.Quote, 0, len(symbols))
	for i, res := range ch {
		if res.err != nil {
			log.Printf("GetQuotes: skipping %q: %v", symbols[i], res.err)
			continue
		}
		quotes = append(quotes, res.q)
	}
	return quotes, nil
}

func (c *Client) Search(ctx context.Context, query string) ([]stock.SearchResult, error) {
	endpoint := "https://query1.finance.yahoo.com/v1/finance/search?" + url.Values{
		"q":           {query},
		"lang":        {"en-US"},
		"region":      {"US"},
		"quotesCount": {"10"},
		"newsCount":   {"0"},
	}.Encode()

	var resp searchResponse
	if err := c.get(ctx, endpoint, &resp); err != nil {
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

func (c *Client) fetchQuote(ctx context.Context, symbol string) (stock.Quote, error) {
	endpoint := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d",
		url.PathEscape(symbol),
	)

	var resp chartResponse
	if err := c.get(ctx, endpoint, &resp); err != nil {
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

func (c *Client) get(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("yahoo finance: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// — DTOs —

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol             string  `json:"symbol"`
				ShortName          string  `json:"shortName"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
			} `json:"meta"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

type searchResponse struct {
	Quotes []struct {
		Symbol    string `json:"symbol"`
		ShortName string `json:"shortname"`
		LongName  string `json:"longname"`
		Exchange  string `json:"exchange"`
		QuoteType string `json:"quoteType"`
	} `json:"quotes"`
}

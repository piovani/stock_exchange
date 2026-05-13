package yahoofinance

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/piovani/stock_exchange/internal/domain/stock"
)

func (c *Client) GetHistoricalQuotes(ctx context.Context, symbol string, from, to time.Time) ([]stock.HistoricalQuote, error) {
	endpoint := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&period1=%d&period2=%d",
		url.PathEscape(symbol),
		from.Unix(),
		to.Unix(),
	)

	var resp histChartResponse
	if err := c.get(ctx, endpoint, &resp); err != nil {
		return nil, err
	}
	if resp.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo finance: %s", resp.Chart.Error.Description)
	}
	if len(resp.Chart.Result) == 0 {
		return nil, fmt.Errorf("%w: %s", stock.ErrNotFound, symbol)
	}

	result := resp.Chart.Result[0]
	if len(result.Timestamp) == 0 || len(result.Indicators.Quote) == 0 {
		return nil, nil
	}

	q := result.Indicators.Quote[0]
	var adjCloses []*float64
	if len(result.Indicators.AdjClose) > 0 {
		adjCloses = result.Indicators.AdjClose[0].AdjClose
	}

	quotes := make([]stock.HistoricalQuote, 0, len(result.Timestamp))
	for i, ts := range result.Timestamp {
		if i >= len(q.Close) || q.Close[i] == nil {
			continue
		}

		hq := stock.HistoricalQuote{
			Symbol:   symbol,
			Exchange: result.Meta.ExchangeName,
			Currency: result.Meta.Currency,
			Date:     time.Unix(ts, 0).UTC().Truncate(24 * time.Hour),
			Close:    *q.Close[i],
		}
		if i < len(q.Open) && q.Open[i] != nil {
			hq.Open = *q.Open[i]
		}
		if i < len(q.High) && q.High[i] != nil {
			hq.High = *q.High[i]
		}
		if i < len(q.Low) && q.Low[i] != nil {
			hq.Low = *q.Low[i]
		}
		if i < len(adjCloses) && adjCloses[i] != nil {
			hq.AdjClose = *adjCloses[i]
		}
		if i < len(q.Volume) && q.Volume[i] != nil {
			hq.Volume = *q.Volume[i]
		}

		// change relative to the previous non-null close
		for j := i - 1; j >= 0; j-- {
			if j < len(q.Close) && q.Close[j] != nil {
				prev := *q.Close[j]
				if prev != 0 {
					hq.ChangeAmount = hq.Close - prev
					hq.ChangePercent = hq.ChangeAmount / prev * 100
				}
				break
			}
		}

		quotes = append(quotes, hq)
	}
	return quotes, nil
}

// histChartResponse unmarshals the Yahoo Finance chart endpoint for daily OHLCV history.
type histChartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency     string `json:"currency"`
				Symbol       string `json:"symbol"`
				ExchangeName string `json:"exchangeName"`
				ShortName    string `json:"shortName"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*int64   `json:"volume"`
				} `json:"quote"`
				AdjClose []struct {
					AdjClose []*float64 `json:"adjclose"`
				} `json:"adjclose"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

package stock

import (
	"context"
	"time"
)

type HistoricalQuote struct {
	Symbol        string
	Exchange      string
	Currency      string
	Date          time.Time
	Open          float64
	High          float64
	Low           float64
	Close         float64
	AdjClose      float64
	Volume        int64
	ChangeAmount  float64
	ChangePercent float64
}

type HistoricalRepository interface {
	UpsertMetadata(ctx context.Context, symbol, name, exchange, currency string) error
	SaveQuotes(ctx context.Context, quotes []HistoricalQuote) error
	GetQuotes(ctx context.Context, symbol string, from, to time.Time) ([]HistoricalQuote, error)
	GetTrackedSymbols(ctx context.Context) ([]string, error)
}

type HistoricalClient interface {
	GetHistoricalQuotes(ctx context.Context, symbol string, from, to time.Time) ([]HistoricalQuote, error)
}

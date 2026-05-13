package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/piovani/stock_exchange/internal/domain/stock"
	"github.com/piovani/stock_exchange/internal/infra/postgres"
	"github.com/piovani/stock_exchange/internal/provider/yahoofinance"
)

// defaultSymbols covers major US, ETF, and B3 tickers — a solid LLM analysis base.
var defaultSymbols = []string{
	// US large caps
	"AAPL", "MSFT", "GOOGL", "AMZN", "META", "TSLA", "NVDA",
	"BRK-B", "JPM", "JNJ", "V", "PG", "MA", "UNH", "HD",
	"MRK", "ABBV", "BAC", "XOM", "LLY",
	// US ETFs
	"SPY", "QQQ", "IWM", "GLD", "TLT", "USO",
	// B3 (Brazilian exchange — Yahoo Finance requires .SA suffix)
	"PETR4.SA", "VALE3.SA", "ITUB4.SA", "BBDC4.SA", "ABEV3.SA",
	"B3SA3.SA", "WEGE3.SA", "RENT3.SA", "RADL3.SA", "MGLU3.SA",
}

func main() {
	symbolsFlag := flag.String("symbols", "", "Comma-separated list of symbols (default: built-in list)")
	dbFlag := flag.String("db", "", "PostgreSQL DSN (default: $DATABASE_URL)")
	workers := flag.Int("workers", 5, "Concurrent fetch workers")
	watch := flag.Bool("watch", false, "Keep running and collect data daily after market close")
	flag.Parse()

	dsn := *dbFlag
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		log.Fatal("DATABASE_URL env var or -db flag is required")
	}

	symbols := defaultSymbols
	if *symbolsFlag != "" {
		symbols = strings.Split(*symbolsFlag, ",")
		for i, s := range symbols {
			symbols[i] = strings.TrimSpace(s)
		}
	}

	ctx := context.Background()

	pool, err := postgres.Open(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	repo := postgres.NewRepository(pool)
	client := yahoofinance.NewClient()

	// always do an initial backfill of the last 12 months
	collect(ctx, client, repo, symbols, *workers)

	if *watch {
		log.Println("watch mode: will collect daily at 23:00 UTC")
		for {
			next := nextRunAt(23, 0)
			log.Printf("next collection at %s", next.Format(time.RFC3339))
			time.Sleep(time.Until(next))
			collect(ctx, client, repo, symbols, *workers)
		}
	}
}

// collect fetches the last 12 months of OHLCV data for each symbol and saves it.
func collect(ctx context.Context, client stock.HistoricalClient, repo stock.HistoricalRepository, symbols []string, workers int) {
	from := time.Now().UTC().AddDate(-1, 0, 0)
	to := time.Now().UTC()

	log.Printf("collecting %d symbols from %s to %s", len(symbols), from.Format("2006-01-02"), to.Format("2006-01-02"))

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, sym := range symbols {
		wg.Add(1)
		sem <- struct{}{}
		go func(symbol string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := collectSymbol(ctx, client, repo, symbol, from, to); err != nil {
				log.Printf("WARN %s: %v", symbol, err)
			}
		}(sym)
	}
	wg.Wait()
	log.Println("collection complete")
}

func collectSymbol(ctx context.Context, client stock.HistoricalClient, repo stock.HistoricalRepository, symbol string, from, to time.Time) error {
	quotes, err := client.GetHistoricalQuotes(ctx, symbol, from, to)
	if err != nil {
		return err
	}
	if len(quotes) == 0 {
		return nil
	}

	// upsert symbol metadata from first quote
	first := quotes[0]
	if err := repo.UpsertMetadata(ctx, symbol, symbol, first.Exchange, first.Currency); err != nil {
		return err
	}

	if err := repo.SaveQuotes(ctx, quotes); err != nil {
		return err
	}

	log.Printf("OK  %s: %d days saved", symbol, len(quotes))
	return nil
}

// nextRunAt returns the next occurrence of HH:MM UTC, always in the future.
func nextRunAt(hour, minute int) time.Time {
	now := time.Now().UTC()
	t := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC)
	if !t.After(now) {
		t = t.Add(24 * time.Hour)
	}
	return t
}

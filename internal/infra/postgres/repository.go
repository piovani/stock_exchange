package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/piovani/stock_exchange/internal/domain/stock"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) UpsertMetadata(ctx context.Context, symbol, name, exchange, currency string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO market_metadata (symbol, name, exchange, currency)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (symbol) DO UPDATE SET
			name             = EXCLUDED.name,
			exchange         = EXCLUDED.exchange,
			currency         = EXCLUDED.currency,
			last_updated_at  = NOW()
	`, symbol, name, exchange, currency)
	return err
}

func (r *Repository) SaveQuotes(ctx context.Context, quotes []stock.HistoricalQuote) error {
	if len(quotes) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, q := range quotes {
		batch.Queue(`
			INSERT INTO stock_quotes_history
				(symbol, date, open, high, low, close, adj_close, volume, change_amount, change_percent)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (symbol, date) DO UPDATE SET
				open           = EXCLUDED.open,
				high           = EXCLUDED.high,
				low            = EXCLUDED.low,
				close          = EXCLUDED.close,
				adj_close      = EXCLUDED.adj_close,
				volume         = EXCLUDED.volume,
				change_amount  = EXCLUDED.change_amount,
				change_percent = EXCLUDED.change_percent
		`,
			q.Symbol, q.Date, q.Open, q.High, q.Low,
			q.Close, q.AdjClose, q.Volume,
			q.ChangeAmount, q.ChangePercent,
		)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range quotes {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) GetQuotes(ctx context.Context, symbol string, from, to time.Time) ([]stock.HistoricalQuote, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT symbol, date, open, high, low, close, adj_close, volume, change_amount, change_percent
		FROM stock_quotes_history
		WHERE symbol = $1 AND date BETWEEN $2 AND $3
		ORDER BY date ASC
	`, symbol, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []stock.HistoricalQuote
	for rows.Next() {
		var q stock.HistoricalQuote
		if err := rows.Scan(
			&q.Symbol, &q.Date,
			&q.Open, &q.High, &q.Low, &q.Close, &q.AdjClose,
			&q.Volume, &q.ChangeAmount, &q.ChangePercent,
		); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (r *Repository) GetTrackedSymbols(ctx context.Context) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT symbol FROM market_metadata ORDER BY symbol`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		symbols = append(symbols, s)
	}
	return symbols, rows.Err()
}

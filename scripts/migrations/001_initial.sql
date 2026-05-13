-- Market metadata: one row per tracked symbol
CREATE TABLE IF NOT EXISTS market_metadata (
    symbol       VARCHAR(20)  PRIMARY KEY,
    name         VARCHAR(255),
    exchange     VARCHAR(30),
    currency     VARCHAR(10),
    first_seen_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Daily OHLCV history for each symbol (last 12 months initially)
CREATE TABLE IF NOT EXISTS stock_quotes_history (
    id             BIGSERIAL    PRIMARY KEY,
    symbol         VARCHAR(20)  NOT NULL REFERENCES market_metadata(symbol),
    date           DATE         NOT NULL,
    open           NUMERIC(20, 6),
    high           NUMERIC(20, 6),
    low            NUMERIC(20, 6),
    close          NUMERIC(20, 6),
    adj_close      NUMERIC(20, 6),
    volume         BIGINT,
    change_amount  NUMERIC(20, 6),
    change_percent NUMERIC(10, 4),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_symbol_date UNIQUE (symbol, date)
);

CREATE INDEX IF NOT EXISTS idx_sqh_symbol      ON stock_quotes_history (symbol);
CREATE INDEX IF NOT EXISTS idx_sqh_date        ON stock_quotes_history (date DESC);
CREATE INDEX IF NOT EXISTS idx_sqh_symbol_date ON stock_quotes_history (symbol, date DESC);

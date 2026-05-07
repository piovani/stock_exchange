package stock

import "fmt"

func ParseExchange(s string) (Exchange, error) {
	switch Exchange(s) {
	case ExchangeUS, ExchangeB3, ExchangeAll:
		return Exchange(s), nil
	default:
		return "", fmt.Errorf("exchange %q not supported; use %q, %q, or %q", s, ExchangeUS, ExchangeB3, ExchangeAll)
	}
}

type Quote struct {
	Symbol        string
	ShortName     string
	Price         float64
	Change        float64
	ChangePercent float64
	Timestamp     int64
}

type SearchResult struct {
	Symbol    string
	ShortName string
	Exchange  string
	Type      string
}

type Symbol struct {
	Ticker   string
	Name     string
	Exchange string
	Type     string
}

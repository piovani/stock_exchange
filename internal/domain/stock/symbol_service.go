package stock

import (
	"context"
	"fmt"
)

type SymbolClient interface {
	ListSymbols(ctx context.Context) ([]Symbol, error)
}

type SymbolService struct {
	clients map[Exchange]SymbolClient
}

func NewSymbolService(clients map[Exchange]SymbolClient) *SymbolService {
	return &SymbolService{clients: clients}
}

func (s *SymbolService) ListSymbols(ctx context.Context, exchange Exchange) ([]Symbol, error) {
	switch exchange {
	case ExchangeUS, ExchangeB3:
		c, ok := s.clients[exchange]
		if !ok {
			return nil, fmt.Errorf("%w: no client configured for exchange %q", ErrInvalidInput, exchange)
		}
		return c.ListSymbols(ctx)
	case ExchangeAll:
		var all []Symbol
		for _, c := range s.clients {
			syms, err := c.ListSymbols(ctx)
			if err != nil {
				continue
			}
			all = append(all, syms...)
		}
		return all, nil
	default:
		return nil, fmt.Errorf("%w: exchange %q not supported", ErrInvalidInput, exchange)
	}
}

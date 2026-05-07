package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound     = errors.New("stock not found")
	ErrInvalidInput = errors.New("invalid input")
)

type quoteClient interface {
	GetQuote(ctx context.Context, symbol string) (Quote, error)
	GetQuotes(ctx context.Context, symbols []string) ([]Quote, error)
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

type Service struct {
	client quoteClient
}

func NewService(client quoteClient) *Service {
	return &Service{client: client}
}

func (s *Service) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, fmt.Errorf("%w: symbol is required", ErrInvalidInput)
	}
	q, err := s.client.GetQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (s *Service) ListQuotes(ctx context.Context, symbols []string) ([]Quote, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("%w: at least one symbol is required", ErrInvalidInput)
	}
	return s.client.GetQuotes(ctx, symbols)
}

func (s *Service) Search(ctx context.Context, query string) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("%w: query is required", ErrInvalidInput)
	}
	return s.client.Search(ctx, query)
}

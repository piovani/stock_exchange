package stock

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("stock not found")

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	q, err := s.repo.GetQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (s *Service) ListQuotes(ctx context.Context, symbols []string) ([]Quote, error) {
	return s.repo.GetQuotes(ctx, symbols)
}

func (s *Service) Search(ctx context.Context, query string) ([]SearchResult, error) {
	return s.repo.Search(ctx, query)
}

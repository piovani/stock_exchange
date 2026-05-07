package stockgrpc

import (
	"context"
	"errors"

	"github.com/piovani/stock_exchange/internal/domain/stock"
	pb "github.com/piovani/stock_exchange/pb/stock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	pb.UnimplementedStockServiceServer
	svc *stock.Service
}

func NewHandler(svc *stock.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetQuote(ctx context.Context, req *pb.GetQuoteRequest) (*pb.GetQuoteResponse, error) {
	if req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}
	q, err := h.svc.GetQuote(ctx, req.Symbol)
	if err != nil {
		if errors.Is(err, stock.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "symbol %q not found", req.Symbol)
		}
		return nil, status.Errorf(codes.Internal, "failed to get quote: %v", err)
	}
	return &pb.GetQuoteResponse{Quote: toProto(q)}, nil
}

func (h *Handler) ListQuotes(ctx context.Context, req *pb.ListQuotesRequest) (*pb.ListQuotesResponse, error) {
	if len(req.Symbols) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one symbol is required")
	}
	quotes, err := h.svc.ListQuotes(ctx, req.Symbols)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list quotes: %v", err)
	}
	out := make([]*pb.Quote, len(quotes))
	for i := range quotes {
		out[i] = toProto(&quotes[i])
	}
	return &pb.ListQuotesResponse{Quotes: out}, nil
}

func (h *Handler) SearchStocks(ctx context.Context, req *pb.SearchStocksRequest) (*pb.SearchStocksResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}
	results, err := h.svc.Search(ctx, req.Query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search: %v", err)
	}
	out := make([]*pb.StockInfo, len(results))
	for i, r := range results {
		out[i] = &pb.StockInfo{
			Symbol:    r.Symbol,
			ShortName: r.ShortName,
			Exchange:  r.Exchange,
			Type:      r.Type,
		}
	}
	return &pb.SearchStocksResponse{Results: out}, nil
}

func toProto(q *stock.Quote) *pb.Quote {
	return &pb.Quote{
		Symbol:        q.Symbol,
		ShortName:     q.ShortName,
		Price:         q.Price,
		Change:        q.Change,
		ChangePercent: q.ChangePercent,
		Timestamp:     q.Timestamp,
	}
}

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
	svc       *stock.Service
	symbolSvc *stock.SymbolService
}

func NewHandler(svc *stock.Service, symbolSvc *stock.SymbolService) *Handler {
	return &Handler{svc: svc, symbolSvc: symbolSvc}
}

func (h *Handler) GetQuote(ctx context.Context, req *pb.GetQuoteRequest) (*pb.GetQuoteResponse, error) {
	q, err := h.svc.GetQuote(ctx, req.Symbol)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &pb.GetQuoteResponse{Quote: toProto(q)}, nil
}

func (h *Handler) ListQuotes(ctx context.Context, req *pb.ListQuotesRequest) (*pb.ListQuotesResponse, error) {
	quotes, err := h.svc.ListQuotes(ctx, req.Symbols)
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*pb.Quote, len(quotes))
	for i := range quotes {
		out[i] = toProto(&quotes[i])
	}
	return &pb.ListQuotesResponse{Quotes: out}, nil
}

func (h *Handler) SearchStocks(ctx context.Context, req *pb.SearchStocksRequest) (*pb.SearchStocksResponse, error) {
	results, err := h.svc.Search(ctx, req.Query)
	if err != nil {
		return nil, toGRPCError(err)
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

func (h *Handler) ListSymbols(ctx context.Context, req *pb.ListSymbolsRequest) (*pb.ListSymbolsResponse, error) {
	exchange, err := stock.ParseExchange(req.Exchange)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	symbols, err := h.symbolSvc.ListSymbols(ctx, exchange)
	if err != nil {
		return nil, toGRPCError(err)
	}
	out := make([]*pb.StockInfo, len(symbols))
	for i, s := range symbols {
		out[i] = &pb.StockInfo{
			Symbol:    s.Ticker,
			ShortName: s.Name,
			Exchange:  s.Exchange,
			Type:      s.Type,
		}
	}
	return &pb.ListSymbolsResponse{Symbols: out, Total: int32(len(out))}, nil
}

func toGRPCError(err error) error {
	switch {
	case errors.Is(err, stock.ErrInvalidInput):
		return status.Errorf(codes.InvalidArgument, "%v", err)
	case errors.Is(err, stock.ErrNotFound):
		return status.Errorf(codes.NotFound, "%v", err)
	default:
		return status.Errorf(codes.Internal, "%v", err)
	}
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

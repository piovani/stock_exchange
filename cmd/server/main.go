package main

import (
	"log"
	"net"
	"os"

	"github.com/piovani/stock_exchange/internal/domain/stock"
	stockgrpc "github.com/piovani/stock_exchange/internal/grpc/stock"
	"github.com/piovani/stock_exchange/internal/infra/yahoofinance"
	pb "github.com/piovani/stock_exchange/pb/stock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	repo := yahoofinance.NewRepository(yahoofinance.NewClient())
	svc := stock.NewService(repo)
	handler := stockgrpc.NewHandler(svc)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterStockServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	log.Printf("gRPC server listening on :%s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

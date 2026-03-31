package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/proto/grpc_service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", appConfig.GrpcPort))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middlewares.AddGrpcLogContext),
		grpc.StreamInterceptor(middlewares.AddGrpcLogContextStream()),
	)
	grpcService := NewGRPCService(
		mediaRepo,
		mediaCaptionRepo,
		cdnFileRepo,
		storageHelper,
		appConfig.InputStoragePath,
		appConfig.OutputStoragePath,
		mediaService,
	)
	grpc_service.RegisterGRPCServiceServer(grpcServer, grpcService)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("grpc.service.v1.GRPCService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	log.Printf("gRPC server listening at %v", lis.Addr())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	sig := <-sigChan
	log.Printf("Received signal: %s. Initiating shutdown...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	grpcServer.GracefulStop()
	log.Println("gRPC server stopped gracefully.")
	<-shutdownCtx.Done()
	log.Println("Shutdown completed.")
}

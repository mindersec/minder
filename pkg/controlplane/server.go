package controlplane

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/spf13/viper"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
)

func loggingInterceptor(level string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			for key, values := range md {
				for _, value := range values {
					log.Printf("[%s] header received: %s=%s", level, key, value)
				}
			}
		}
		resp, err := handler(ctx, req)
		log.Printf("[%s] method called: %s", level, info.FullMethod)
		log.Printf("[%s] incoming request: %v", level, req)

		log.Printf("[%s] outgoing response: %v", level, resp)
		return resp, err
	}
}

func StartGRPCServer(address string) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Initializing logger in level: " + viper.GetString("logging.level"))

	var s *grpc.Server

	if viper.GetString("logging.level") == "debug" {
		s = grpc.NewServer(
			grpc.Creds(insecure.NewCredentials()),
			grpc.UnaryInterceptor(loggingInterceptor(viper.GetString("logging.level"))),
		)
	} else {
		s = grpc.NewServer(
			grpc.Creds(insecure.NewCredentials()),
		)
	}

	// register the services (declared within register_handlers.go)
	RegisterGRPCServices(s)

	reflection.Register(s)

	log.Printf("Starting gRPC server on %s", address)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func StartHTTPServer(address, grpcAddress string) {

	mux := http.NewServeMux()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gwmux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// register the services (declared within register_handlers.go)
	RegisterGatewayHTTPHandlers(ctx, gwmux, grpcAddress, opts)

	mux.Handle("/", gwmux)

	log.Printf("Starting HTTP server on %s", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-serves/rest"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	apiv1 "github.com/go-serves/rest/example/gen/go/proto/api/v1"
)

// userService implements the generated gRPC interface apiv1.UserServiceServer
type userService struct {
	apiv1.UnimplementedUserServiceServer
}

// GetUser returns a user by ID
func (s *userService) GetUser(ctx context.Context, req *apiv1.GetUserRequest) (*apiv1.GetUserResponse, error) {
	return &apiv1.GetUserResponse{
		Id:   req.GetId(),
		Name: fmt.Sprintf("User-%s", req.GetId()),
	}, nil
}

func (s *userService) RegisterGateway(ctx context.Context, mux *runtime.ServeMux) error {
	return apiv1.RegisterUserServiceHandlerServer(ctx, mux, s)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()

	config, err := rest.LoadConfig("test")
	if err != nil {
		logger.Error("config loading failed", "err", err)
		os.Exit(1)
	}

	server, err := rest.NewServer(ctx, config, &userService{}, logger)
	if err != nil {
		logger.Error("server instantiation failed", "err", err)
		os.Exit(1)
	}

	if err := server.Run(); err != nil {
		logger.Error("server startup failed", "err", err)
		os.Exit(1)
	}
}

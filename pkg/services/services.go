package services

import (
	"context"
	"fmt"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type Server struct {
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedAuthUrlServiceServer
	OAuth2       *oauth2.Config
	ClientID     string
	ClientSecret string
}

func (s *Server) CheckHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "OK"}, nil
}

func NewServer(oauth2Config *oauth2.Config) *Server {
	return &Server{
		OAuth2: oauth2Config,
	}
}

func (s *Server) AuthUrl(ctx context.Context, req *pb.AuthUrlRequest) (*pb.AuthUrlResponse, error) {
	oauth2Cfg := &oauth2.Config{
		ClientID:     s.ClientID,
		ClientSecret: s.ClientSecret,
		Endpoint:     github.Endpoint,
		RedirectURL:  "http://localhost:8080/api/v1/callback",
		Scopes:       []string{"user:email"},
	}
	url := oauth2Cfg.AuthCodeURL("state")
	fmt.Println("URL:", url)
	response := &pb.AuthUrlResponse{
		Url: url,
	}
	return response, nil
}

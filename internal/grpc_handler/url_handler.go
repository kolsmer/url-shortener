package grpc_handler

import (
	"context"
	"errors"
	pb "url-shortener/internal/proto"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type URLGRPCHandler struct {
	pb.UnimplementedURLServiceServer
	service *service.URLService
}

func NewURLGRPCHandler(svc *service.URLService) *URLGRPCHandler {
	return &URLGRPCHandler{
		service: svc,
	}
}

func (h *URLGRPCHandler) ShortenURL(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	code,err := h.service.ShortenURL(ctx, req.GetUrl())
	if errors.Is(err, storage.ErrEmptyURL) || errors.Is(err, service.ErrInvalidURL) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	
	}
	return &pb.ShortenResponse{
			Code: code,
			ShortCode: "http://localhost:8080/" + code,
		}, nil
}

func (h *URLGRPCHandler) GetOriginalURL(ctx context.Context, req *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	url, err := h.service.GetOriginalURL(ctx, req.GetCode())
	if errors.Is(err, storage.ErrURLNotFound) {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if err != nil {	
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetURLResponse{OriginalUrl: url}, nil

}
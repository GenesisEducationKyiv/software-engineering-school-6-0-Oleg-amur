package grpcapi

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/api/grpc/pb"
	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/api/http/dto"
	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/apperr"
	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GrpcHandler struct {
	pb.UnimplementedReleaseNotifierServer
	log     *slog.Logger
	service *service.SubscriptionService
}

func NewGrpcHandler(log *slog.Logger, svc *service.SubscriptionService) *GrpcHandler {
	return &GrpcHandler{
		log:     log,
		service: svc,
	}
}

func (h *GrpcHandler) Subscribe(ctx context.Context, req *pb.SubscribeRequest) (*pb.SubscribeResponse, error) {
	err := h.service.Subscribe(ctx, dto.SubscribeRequest{
		Email: req.GetEmail(),
		Repo:  req.GetRepo(),
	})
	if err != nil {
		if errors.Is(err, apperr.ErrInvalidFormat) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, apperr.ErrRepoNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if errors.Is(err, apperr.ErrRateLimitExceeded) {
			return nil, status.Error(codes.ResourceExhausted, err.Error())
		}
		if errors.Is(err, apperr.ErrAlreadySubscribed) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		h.log.Error("subscription failed", "err", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	return &pb.SubscribeResponse{}, nil
}

func (h *GrpcHandler) Confirm(ctx context.Context, req *pb.ConfirmRequest) (*pb.ConfirmResponse, error) {
	err := h.service.Confirm(ctx, req.GetToken())
	if err != nil {
		if errors.Is(err, apperr.ErrTokenNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		h.log.Error("confirmation failed", "err", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	return &pb.ConfirmResponse{}, nil
}

func (h *GrpcHandler) Unsubscribe(ctx context.Context, req *pb.UnsubscribeRequest) (*pb.UnsubscribeResponse, error) {
	err := h.service.Unsubscribe(ctx, req.GetToken())
	if err != nil {
		h.log.Error("unsubscription failed", "err", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	return &pb.UnsubscribeResponse{}, nil
}

func (h *GrpcHandler) GetSubscriptions(ctx context.Context, req *pb.GetSubscriptionsRequest) (*pb.GetSubscriptionsResponse, error) {
	subs, err := h.service.GetSubscriptions(ctx, req.GetEmail())
	if err != nil {
		h.log.Error("get subscriptions failed", "err", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	var pbSubs []*pb.Subscription
	for _, s := range subs {
		pbSubs = append(pbSubs, &pb.Subscription{
			Email:       s.Email,
			Repo:        s.Repo,
			Confirmed:   s.Confirmed,
			LastSeenTag: s.LastSeenTag,
		})
	}

	return &pb.GetSubscriptionsResponse{
		Subscriptions: pbSubs,
	}, nil
}

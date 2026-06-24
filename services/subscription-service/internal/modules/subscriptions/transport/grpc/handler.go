package subscriptiongrpc

import (
	"context"
	"errors"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/api/grpc/pb"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SubscriptionUsecases interface {
	Subscribe(context.Context, subscriptionusecase.SubscribeRequest) error
	Confirm(context.Context, string) error
	Unsubscribe(context.Context, string) error
	GetSubscriptions(context.Context, string) ([]subscriptionusecase.SubscriptionView, error)
}

type Handler struct {
	pb.UnimplementedSubscriptionServiceServer
	log      *slog.Logger
	usecases SubscriptionUsecases
}

func NewHandler(log *slog.Logger, usecases SubscriptionUsecases) *Handler {
	return &Handler{
		log:      log,
		usecases: usecases,
	}
}

func (h *Handler) Subscribe(
	ctx context.Context,
	req *pb.SubscribeRequest,
) (*pb.SubscribeResponse, error) {
	subReq := subscriptionusecase.SubscribeRequest{
		Email: req.GetEmail(),
		Repo:  req.GetRepo(),
	}

	if err := subReq.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := h.usecases.Subscribe(ctx, subReq)
	if err != nil {
		if errors.Is(err, apperr.ErrInvalidEmailFormat) ||
			errors.Is(err, apperr.ErrInvalidRepositoryFormat) {
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

func (h *Handler) Confirm(
	ctx context.Context,
	req *pb.ConfirmRequest,
) (*pb.ConfirmResponse, error) {
	err := h.usecases.Confirm(ctx, req.GetToken())
	if err != nil {
		if errors.Is(err, apperr.ErrTokenNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		h.log.Error("confirmation failed", "err", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	return &pb.ConfirmResponse{}, nil
}

func (h *Handler) Unsubscribe(
	ctx context.Context,
	req *pb.UnsubscribeRequest,
) (*pb.UnsubscribeResponse, error) {
	err := h.usecases.Unsubscribe(ctx, req.GetToken())
	if err != nil {
		h.log.Error("unsubscription failed", "err", err)
		return nil, status.Error(codes.Internal, "Internal server error")
	}

	return &pb.UnsubscribeResponse{}, nil
}

func (h *Handler) GetSubscriptions(
	ctx context.Context,
	req *pb.GetSubscriptionsRequest,
) (*pb.GetSubscriptionsResponse, error) {
	subs, err := h.usecases.GetSubscriptions(ctx, req.GetEmail())
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

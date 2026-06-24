package subscriptiongrpc

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/usecase"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/gen/subscriptions/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SubscriptionUsecases interface {
	Subscribe(context.Context, subscriptionusecase.SubscribeRequest) error
	Confirm(context.Context, string) error
	Unsubscribe(context.Context, string) error
	GetSubscriptions(context.Context, string) ([]subscriptionusecase.SubscriptionView, error)
	GetActiveSubscriptionsByRepository(context.Context, string) ([]domain.RepositorySubscription, error)
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

func (h *Handler) ListActiveSubscriptionsByRepository(
	ctx context.Context,
	req *pb.ListActiveSubscriptionsByRepositoryRequest,
) (*pb.ListActiveSubscriptionsByRepositoryResponse, error) {
	repository := strings.TrimSpace(req.GetRepository())
	if repository == "" {
		return nil, status.Error(codes.InvalidArgument, "repository is required")
	}

	subscriptions, err := h.usecases.GetActiveSubscriptionsByRepository(ctx, repository)
	if err != nil {
		h.log.Error("get active subscriptions by repository failed", "repository", repository, "err", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	response := &pb.ListActiveSubscriptionsByRepositoryResponse{
		Subscriptions: make([]*pb.ActiveSubscription, 0, len(subscriptions)),
	}
	for _, subscription := range subscriptions {
		response.Subscriptions = append(response.Subscriptions, &pb.ActiveSubscription{
			Email:            subscription.Email,
			UnsubscribeToken: subscription.UnsubscribeToken,
		})
	}

	return response, nil
}

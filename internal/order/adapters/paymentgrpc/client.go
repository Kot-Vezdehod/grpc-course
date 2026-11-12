package paymentgrpc

import (
	"context"
	"errors"
	paymentpb "grpc-course/gen/payment"
	"grpc-course/internal/order/domain"
	"log"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Client struct {
	client paymentpb.PaymentServiceClient
	cb     *gobreaker.CircuitBreaker
}

func NewClient(client paymentpb.PaymentServiceClient) *Client {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "payment_service",
		MaxRequests: 1,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("circuit breaker %s: %s -> %s", name, from.String(), to.String())
		},
		IsSuccessful: func(err error) bool {
			return !isBreakerFailure(err)
		},
	})

	return &Client{
		client: client,
		cb:     cb,
	}
}

func (c *Client) CreatePayment(
	ctx context.Context,
	orderID string,
	amountMinor int64,
	currency string,
) error {
	req := paymentpb.CreatePaymentRequest{
		OrderId:     orderID,
		AmountMinor: amountMinor,
		Currency:    currency,
	}

	_, err := c.cb.Execute(func() (interface{}, error) {
		_, err := c.client.CreatePayment(ctx, &req)
		if err != nil {
			return nil, err
		}

		return nil, nil
	})

	if err != nil {
		log.Printf("raw payment error: %v", err)

		if errors.Is(err, gobreaker.ErrOpenState) {
			return domain.ErrPaymentUnavailable
		}

		switch status.Code(err) {
		case codes.Unavailable,
			codes.ResourceExhausted:
			return domain.ErrPaymentUnavailable
		case codes.DeadlineExceeded:
			return domain.ErrPaymentTimeout
		}

		return err
	}

	return nil
}

func isBreakerFailure(err error) bool {
	if err == nil {
		return false
	}

	switch status.Code(err) {
	case codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

package service

import (
	"context"
	"errors"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/accrual"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/rs/zerolog"
)

// AccrualWorker polls orders and updates their status/balance.
type AccrualWorker struct {
	orderService   *OrderService
	balanceService *BalanceService
	accrualClient  *accrual.Client
	logger         zerolog.Logger
	interval       time.Duration
}

// NewAccrualWorker creates a new worker.
func NewAccrualWorker(
	orderService *OrderService,
	balanceService *BalanceService,
	accrualClient *accrual.Client,
	logger zerolog.Logger,
	interval time.Duration,
) *AccrualWorker {
	return &AccrualWorker{
		orderService:   orderService,
		balanceService: balanceService,
		accrualClient:  accrualClient,
		logger:         logger,
		interval:       interval,
	}
}

// Run starts the background polling loop.
func (w *AccrualWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Info().Msg("accrual worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("accrual worker stopped")
			return
		case <-ticker.C:
			w.processOrders(ctx)
		}
	}
}

func (w *AccrualWorker) processOrders(ctx context.Context) {
	// Process NEW orders first
	newOrders, err := w.orderService.GetNewOrders(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to get new orders")
		return
	}
	for _, order := range newOrders {
		w.checkOrder(ctx, order)
	}

	// Process PROCESSING orders
	processingOrders, err := w.orderService.GetProcessingOrders(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to get processing orders")
		return
	}
	for _, order := range processingOrders {
		w.checkOrder(ctx, order)
	}
}

func (w *AccrualWorker) checkOrder(ctx context.Context, order *domain.Order) {
	resp, err := w.accrualClient.GetOrderStatusWithRetry(ctx, order.Number)
	if err != nil {
		if errors.Is(err, accrual.ErrTooManyRequests) {
			w.logger.Warn().Msg("accrual system rate limit hit, waiting")
			time.Sleep(time.Minute) // Simple backoff
			return
		}
		w.logger.Error().Err(err).Str("order", order.Number).Msg("failed to get accrual status")
		return
	}

	if resp == nil {
		// 204 No Content - order not yet registered in accrual system
		// Keep status as NEW or PROCESSING depending on initial state
		// If it was NEW, we might want to move it to PROCESSING to indicate we are trying
		if order.Status == domain.OrderStatusNew {
			err = w.orderService.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusProcessing, nil)
			if err != nil {
				w.logger.Error().Err(err).Str("order", order.Number).Msg("failed to update order status to PROCESSING")
			}
		}
		return
	}

	// Map accrual status to internal status
	var newStatus domain.OrderStatus
	var accrualAmount *float64

	switch resp.Status {
	case "REGISTERED", "PROCESSING":
		newStatus = domain.OrderStatusProcessing
	case "PROCESSED":
		newStatus = domain.OrderStatusProcessed
		accrualAmount = resp.Accrual
	case "INVALID":
		newStatus = domain.OrderStatusInvalid
	default:
		w.logger.Warn().Str("status", resp.Status).Str("order", order.Number).Msg("unknown accrual status")
		return
	}

	// Update order status
	err = w.orderService.UpdateOrderStatus(ctx, order.ID, newStatus, accrualAmount)
	if err != nil {
		w.logger.Error().Err(err).Str("order", order.Number).Msg("failed to update order status")
		return
	}

	// If processed, credit balance
	if newStatus == domain.OrderStatusProcessed && accrualAmount != nil {
		err = w.balanceService.CreditOrder(ctx, order.Number, *accrualAmount)
		if err != nil {
			w.logger.Error().Err(err).Str("order", order.Number).Float64("accrual", *accrualAmount).Msg("failed to credit balance")
			// Note: This is critical. In production, you'd want a retry mechanism or transactional integrity here.
		} else {
			w.logger.Info().Str("order", order.Number).Float64("accrual", *accrualAmount).Msg("balance credited")
		}
	}
}

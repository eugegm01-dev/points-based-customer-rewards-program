package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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
	cooldownUntil  atomic.Value // time.Time
	cooldownMu     sync.RWMutex
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
	orders, err := w.getAllPendingOrders(ctx) // NEW + PROCESSING
	if err != nil {
		return
	}
	if len(orders) == 0 {
		return
	}

	const workerCount = 5 // настраиваемый параметр
	jobs := make(chan *domain.Order, len(orders))
	var wg sync.WaitGroup

	// Запуск воркеров
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for order := range jobs {
				w.checkOrder(ctx, order)
			}
		}()
	}

	// Отправка задач
	for _, order := range orders {
		select {
		case jobs <- order:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		}
	}
	close(jobs)
	wg.Wait()
}

// getAllPendingOrders собирает заказы со статусами NEW и PROCESSING.
func (w *AccrualWorker) getAllPendingOrders(ctx context.Context) ([]*domain.Order, error) {
	newOrders, err := w.orderService.GetNewOrders(ctx)
	if err != nil {
		return nil, err
	}
	processingOrders, err := w.orderService.GetProcessingOrders(ctx)
	if err != nil {
		return nil, err
	}
	return append(newOrders, processingOrders...), nil
}

// checkOrder проверяет статус заказа в accrual системе.
func (w *AccrualWorker) checkOrder(ctx context.Context, order *domain.Order) {
	// ✅ Проверка глобального кулдауна
	if until, ok := w.cooldownUntil.Load().(time.Time); ok {
		if w.waitIfCooldown(ctx) {
			return // graceful shutdown
		}
		if sleep := time.Until(until); sleep > 0 {
			select {
			case <-time.After(sleep): // ✅ прерываемый сон
			case <-ctx.Done(): // ✅ graceful shutdown
				return
			}
		}
	}

	// ✅ 2. Запрос к accrual (ретраи уже внутри клиента)
	resp, err := w.accrualClient.GetOrderStatus(ctx, order.Number)

	// ✅ 3. Обработка ошибок
	if err != nil {
		var rlErr *accrual.RateLimitError
		if errors.As(err, &rlErr) {
			w.setGlobalCooldown(rlErr.RetryAfter) // ✅ используем реальное значение
			return
		}
		w.logger.Error().Err(err).Str("order", order.Number).Msg("failed to get accrual status")
		return
	}

	// ✅ 4. Обработка 204 No Content
	if resp == nil {
		if order.Status == domain.OrderStatusNew {
			_ = w.orderService.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusProcessing, nil)
		}
		return
	}

	// ✅ 5. Маппинг статусов
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

	// ✅ 6. Обновление статуса заказа
	_ = w.orderService.UpdateOrderStatus(ctx, order.ID, newStatus, accrualAmount)

	// ✅ 7. Кредитование баланса (в транзакции)
	if newStatus == domain.OrderStatusProcessed && accrualAmount != nil {
		if err := w.balanceService.CreditOrder(ctx, order.Number, *accrualAmount); err != nil {
			w.logger.Error().Err(err).Str("order", order.Number).Float64("accrual", *accrualAmount).Msg("failed to credit balance")
		} else {
			w.logger.Info().Str("order", order.Number).Float64("accrual", *accrualAmount).Msg("balance credited")
		}
	}
}

// ✅ Вспомогательный метод: ожидание кулдауна с поддержкой graceful shutdown
func (w *AccrualWorker) waitIfCooldown(ctx context.Context) bool {
	if until, ok := w.cooldownUntil.Load().(time.Time); ok {
		if sleep := time.Until(until); sleep > 0 {
			w.logger.Debug().Dur("sleep", sleep).Msg("waiting due to global cooldown")
			select {
			case <-time.After(sleep):
				// продолжаем работу
			case <-ctx.Done():
				w.logger.Info().Msg("worker stopped during cooldown")
				return true // сигнал о завершении
			}
		}
	}
	return false
}

// setGlobalCooldown устанавливает глобальный кулдаун для всех воркеров.
func (w *AccrualWorker) setGlobalCooldown(duration time.Duration) {
	w.cooldownMu.Lock()
	defer w.cooldownMu.Unlock()
	until := time.Now().Add(duration)
	w.cooldownUntil.Store(until)
	w.logger.Warn().Time("until", until).Msg("global cooldown set")
}

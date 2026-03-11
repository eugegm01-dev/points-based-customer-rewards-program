package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/accrual"
	"github.com/eugegm01-dev/points-based-customer-rewards-program.git/internal/domain"
	"github.com/rs/zerolog"
)

// WorkerPool управляет фиксированным количеством воркеров, обрабатывающих заказы.
type WorkerPool struct {
	numWorkers    int
	taskCh        chan *domain.Order
	logger        zerolog.Logger
	orderSvc      *OrderService
	balanceSvc    *BalanceService
	accrualClient *accrual.Client

	// для паузы
	pauseMu    sync.RWMutex
	paused     bool
	pauseUntil time.Time
	resumeCh   chan struct{} // закрывается при возобновлении работы

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewWorkerPool создаёт новый пул.
func NewWorkerPool(
	numWorkers int,
	orderSvc *OrderService,
	balanceSvc *BalanceService,
	accrualClient *accrual.Client,
	logger zerolog.Logger,
) *WorkerPool {
	return &WorkerPool{
		numWorkers:    numWorkers,
		taskCh:        make(chan *domain.Order, 100),
		logger:        logger,
		orderSvc:      orderSvc,
		balanceSvc:    balanceSvc,
		accrualClient: accrualClient,
		resumeCh:      make(chan struct{}),
	}
}

// Start запускает воркеры и генератор заданий.
func (p *WorkerPool) Start(ctx context.Context) {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// запуск воркеров
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	// запуск генератора
	p.wg.Add(1)
	go p.generator()

	p.logger.Info().Int("workers", p.numWorkers).Msg("worker pool started")
}

// Stop останавливает пул и ожидает завершения всех горутин.
func (p *WorkerPool) Stop() {
	p.cancel()
	p.wg.Wait()
	p.logger.Info().Msg("worker pool stopped")
}

// worker – основной цикл обработки заказов.
func (p *WorkerPool) worker() {
	defer p.wg.Done()
	for {
		// Проверка паузы перед взятием нового заказа
		if err := p.waitIfPaused(); err != nil {
			return // контекст отменён
		}

		select {
		case <-p.ctx.Done():
			return
		case order := <-p.taskCh:
			p.checkOrder(order)
		}
	}
}

// waitIfPaused блокирует воркер, пока активна пауза.
// Возвращает ошибку, если контекст отменён.
func (p *WorkerPool) waitIfPaused() error {
	for {
		p.pauseMu.RLock()
		if !p.paused {
			p.pauseMu.RUnlock()
			return nil
		}
		until := p.pauseUntil
		p.pauseMu.RUnlock()

		if now := time.Now(); now.After(until) {
			// попытаться снять паузу (если время истекло)
			p.pauseMu.Lock()
			if p.paused && time.Now().After(p.pauseUntil) {
				p.paused = false
			}
			p.pauseMu.Unlock()
			continue
		}

		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// checkOrder обрабатывает один заказ.
func (p *WorkerPool) checkOrder(order *domain.Order) {
	// запрос к accrual
	resp, err := p.accrualClient.GetOrderStatus(p.ctx, order.Number)
	if err != nil {
		var rlErr *accrual.RateLimitError
		if errors.As(err, &rlErr) {
			p.pause(rlErr.RetryAfter) // глобальная пауза
			return
		}
		p.logger.Error().Err(err).Str("order", order.Number).Msg("failed to get accrual status")
		return
	}

	// 204 No Content – заказ ещё не зарегистрирован в accrual
	if resp == nil {
		if order.Status == domain.OrderStatusNew {
			_ = p.orderSvc.UpdateOrderStatus(p.ctx, order.ID, domain.OrderStatusProcessing, nil)
		}
		return
	}

	// маппинг статусов
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
		p.logger.Warn().Str("status", resp.Status).Str("order", order.Number).Msg("unknown accrual status")
		return
	}

	// обновление статуса заказа
	_ = p.orderSvc.UpdateOrderStatus(p.ctx, order.ID, newStatus, accrualAmount)

	// кредитование баланса для PROCESSED
	if newStatus == domain.OrderStatusProcessed && accrualAmount != nil {
		if err := p.balanceSvc.CreditOrder(p.ctx, order.Number, *accrualAmount); err != nil {
			p.logger.Error().Err(err).Str("order", order.Number).Float64("accrual", *accrualAmount).Msg("failed to credit balance")
		} else {
			p.logger.Info().Str("order", order.Number).Float64("accrual", *accrualAmount).Msg("balance credited")
		}
	}
}

// pause активирует глобальную паузу на указанную длительность.
// Если пауза уже активна, время продлевается до максимума.
func (p *WorkerPool) pause(duration time.Duration) {
	p.pauseMu.Lock()
	defer p.pauseMu.Unlock()

	now := time.Now()
	newUntil := now.Add(duration)

	if !p.paused || newUntil.After(p.pauseUntil) {
		p.pauseUntil = newUntil
	}
	p.paused = true
	p.logger.Warn().Time("until", p.pauseUntil).Msg("global cooldown set")
}

// generator периодически опрашивает БД и отправляет заказы в канал.
func (p *WorkerPool) generator() {
	defer p.wg.Done()
	ticker := time.NewTicker(10 * time.Second) // настраиваемый интервал
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.fetchAndEnqueue()
		}
	}
}

// fetchAndEnqueue получает новые и обрабатываемые заказы и отправляет их в канал.
func (p *WorkerPool) fetchAndEnqueue() {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	newOrders, err := p.orderSvc.GetNewOrders(ctx)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to fetch new orders")
		return
	}
	processingOrders, err := p.orderSvc.GetProcessingOrders(ctx)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to fetch processing orders")
		return
	}

	allOrders := append(newOrders, processingOrders...)
	for _, order := range allOrders {
		select {
		case p.taskCh <- order:
		case <-ctx.Done():
			return
		}
	}
}

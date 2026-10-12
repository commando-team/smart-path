package router

import (
	"cmp"
	"slices"

	"github.com/commando-team/smart-path/internal/domain"
	"github.com/google/uuid"
)

const (
	WeightFee     = 100
	WeightLatency = 1
	WeightFailure = 200
)

var DefaultChannels = []*domain.Channel{
	{
		Id:             uuid.New(),
		Name:           "T-Bank",
		IsActive:       true,
		FeePercent:     0,
		SuccessPercent: 99,
		LatencyMs:      100,
	},
	{
		Id:             uuid.New(),
		Name:           "Sberbank",
		IsActive:       true,
		FeePercent:     10,
		SuccessPercent: 80,
		LatencyMs:      500,
	},
}

func GetBestChannel(channels []*domain.Channel) *domain.Channel {
	var activeChannels []*domain.Channel

	for _, channel := range channels {
		if channel.IsActive {
			activeChannels = append(activeChannels, channel)
		}
	}

	if len(activeChannels) == 0 {
		return nil
	}

	slices.SortFunc(activeChannels, func(a, b *domain.Channel) int {
		return cmp.Compare(calculateScore(a), calculateScore(b))
	})

	return activeChannels[0]
}

func calculateScore(channel *domain.Channel) int64 {
	// channel.mu.RLock()
	// defer channel.mu.RUnlock()

	failurePercent := 100 - channel.SuccessPercent

	return (channel.FeePercent * WeightFee) +
		(channel.LatencyMs * WeightLatency) +
		(failurePercent * WeightFailure)
}

// type Channel struct {
// 	mu sync.RWMutex

// 	Id         uuid.UUID
// 	Name       string
// 	IsActive   bool
// 	FeePercent int64 // Базовая комиссия (не меняется динамически)

// 	// Динамические метрики (меняются на лету)
// 	LatencyMs      int64 // Экспоненциальное скользящее среднее
// 	SuccessPercent int64 // Динамический % успеха

// 	// Счетчики для расчета метрик
// 	totalRequests    int64
// 	successRequests  int64
// 	consecutiveFails int64 // Счетчик ошибок ПОДРЯД (для Circuit Breaker)

// 	// Circuit Breaker (Предохранитель)
// 	IsCircuitOpen   bool      // Если true - канал "выбит"
// 	CircuitCooldown time.Time // До какого момента канал отключен
// }

// func (c *Channel) IsAvailable() bool {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	if !c.IsActive {
// 		return false
// 	}

// 	// Если предохранитель выбит, проверяем — не истек ли кулдаун
// 	if c.IsCircuitOpen {
// 		if time.Now().After(c.CircuitCooldown) {
// 			// Кулдаун прошел -> даем каналу шанс (Half-Open)
// 			c.IsCircuitOpen = false
// 			c.consecutiveFails = 0
// 			return true
// 		}
// 		return false // Канал всё еще отдыхает
// 	}

// 	return true
// }

// const (
// 	MaxConsecutiveFails = 3                // После 3 ошибок подряд выбиваем предохранитель
// 	CircuitCooldownTime = 30 * time.Second // Канал отключается на 30 секунд
// )

// // RecordSuccess вызывается, когда платеж прошел успешно
// func (c *Channel) RecordSuccess(latency time.Duration) {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	c.totalRequests++
// 	c.successRequests++
// 	c.consecutiveFails = 0 // сбрасываем счетчик подряд идущих ошибок

// 	// 1. Пересчитываем динамический процент успеха
// 	c.SuccessPercent = (c.successRequests * 100) / c.totalRequests

// 	// 2. Сглаживаем задержку: 80% старого значения + 20% нового замера
// 	currentLatencyMs := latency.Milliseconds()
// 	if c.LatencyMs == 0 {
// 		c.LatencyMs = currentLatencyMs
// 	} else {
// 		c.LatencyMs = (c.LatencyMs*4 + currentLatencyMs) / 5
// 	}
// }

// // RecordFailure вызывается при техническом сбое канала или таймауте
// func (c *Channel) RecordFailure() {
// 	c.mu.Lock()
// 	defer c.mu.Unlock()

// 	c.totalRequests++
// 	c.consecutiveFails++

// 	// 1. Пересчитываем динамический процент успеха
// 	c.SuccessPercent = (c.successRequests * 100) / c.totalRequests

// 	// 2. Логика Circuit Breaker
// 	if c.consecutiveFails >= MaxConsecutiveFails {
// 		c.IsCircuitOpen = true
// 		c.CircuitCooldown = time.Now().Add(CircuitCooldownTime)
// 	}
// }

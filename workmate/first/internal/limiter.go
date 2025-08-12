package internal

import "errors"

var ErrLimit = errors.New("err limit")

// ограничение одновременых запросов
type RateLimiter struct {
	lim chan struct{}
}

// Создание лимитера
func NewRateLimiter(maxConcurrent int) *RateLimiter {
	return &RateLimiter{
		lim: make(chan struct{}, maxConcurrent),
	}
}

// Занимаем слот если есть свободный
func (rl *RateLimiter) TryAcquire() error {
	select {
	case rl.lim <- struct{}{}:
		return nil
	default:
		return ErrLimit
	}
}

// Занимаем слот
func (rl *RateLimiter) Acquire() {
	rl.lim <- struct{}{}

}

// освобождаем слот
func (rl *RateLimiter) Release() {
	<-rl.lim
}

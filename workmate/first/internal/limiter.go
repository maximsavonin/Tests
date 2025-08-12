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

// Занимаем слот
func (rl *RateLimiter) Acquire() error {
	select {
	case rl.lim <- struct{}{}:
		return nil
	default:
		return ErrLimit
	}
}

// освобождаем слот
func (rl *RateLimiter) Release() {
	<-rl.lim
}

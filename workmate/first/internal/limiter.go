package internal

// RateLimiter ограничивает количество одновременных запросов
type RateLimiter struct {
	sem chan struct{}
}

// NewRateLimiter создает новый лимитер
func NewRateLimiter(maxConcurrent int) *RateLimiter {
	return &RateLimiter{
		sem: make(chan struct{}, maxConcurrent),
	}
}

// Acquire получает слот (блокирует при достижении лимита)
func (rl *RateLimiter) Acquire() {
	rl.sem <- struct{}{}
}

// Release освобождает слот
func (rl *RateLimiter) Release() {
	<-rl.sem
}

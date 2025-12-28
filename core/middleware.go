package core

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"
)

// Middleware 中间件类型
// 类似于 Gin 的 HandlerFunc，但支持泛型
// ctx: 上下文
// input: 输入数据
// next: 下一个处理器（调用它来传递控制权）
type Middleware[I any, O any] func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error)

// Chain 中间件链
type Chain[I any, O any] struct {
	middlewares []Middleware[I, O]
	final       InvokeFunc[I, O] // 最终的处理函数
}

// NewChain 创建新的中间件链
func NewChain[I any, O any](final InvokeFunc[I, O], middlewares ...Middleware[I, O]) *Chain[I, O] {
	return &Chain[I, O]{
		middlewares: middlewares,
		final:       final,
	}
}

// Use 添加中间件到链中（返回新的链）
func (c *Chain[I, O]) Use(middlewares ...Middleware[I, O]) *Chain[I, O] {
	newMiddlewares := make([]Middleware[I, O], len(c.middlewares)+len(middlewares))
	copy(newMiddlewares, c.middlewares)
	copy(newMiddlewares[len(c.middlewares):], middlewares)
	return &Chain[I, O]{
		middlewares: newMiddlewares,
		final:       c.final,
	}
}

// Execute 执行中间件链
// 按顺序执行中间件，每个中间件可以选择是否调用 next
func (c *Chain[I, O]) Execute(ctx context.Context, input I) (O, error) {
	// 构建处理器链
	handler := c.buildChain(0)

	return handler(ctx, input)
}

// buildChain 递归构建处理器链
func (c *Chain[I, O]) buildChain(index int) InvokeFunc[I, O] {
	// 如果已经到达最后一个中间件，返回最终的处理器
	if index >= len(c.middlewares) {
		return func(ctx context.Context, input I) (O, error) {
			return c.final(ctx, input)
		}
	}

	// 当前中间件
	currentMiddleware := c.middlewares[index]
	// 下一个处理器
	nextHandler := c.buildChain(index + 1)

	// 返回包装后的处理器
	return func(ctx context.Context, input I) (O, error) {
		return currentMiddleware(ctx, input, nextHandler)
	}
}

// LambdaWithMiddleware 支持中间件的 Lambda
type LambdaWithMiddleware[I any, O any] struct {
	chain  *Chain[I, O]
	name   string
	meta   *LambdaMeta
	metrics *LambdaMetrics
}

// NewLambdaWithMiddleware 创建支持中间件的 Lambda
func NewLambdaWithMiddleware[I any, O any](name string, handler InvokeFunc[I, O], middlewares ...Middleware[I, O]) *LambdaWithMiddleware[I, O] {
	chain := NewChain(handler, middlewares...)

	return &LambdaWithMiddleware[I, O]{
		chain:  chain,
		name:   name,
		metrics: &LambdaMetrics{},
	}
}

// Invoke 调用 lambda（执行完整的中间件链）
func (l *LambdaWithMiddleware[I, O]) Invoke(ctx context.Context, input I) (*LambdaResult[O], error) {
	start := time.Now()
	result := &LambdaResult[O]{
		Timestamp: start,
	}

	output, err := l.chain.Execute(ctx, input)

	result.Duration = time.Since(start)
	result.Output = output
	result.Error = err

	return result, err
}

// Use 添加中间件（返回新的 Lambda）
func (l *LambdaWithMiddleware[I, O]) Use(middlewares ...Middleware[I, O]) *LambdaWithMiddleware[I, O] {
	newChain := l.chain.Use(middlewares...)
	return &LambdaWithMiddleware[I, O]{
		chain:   newChain,
		name:    l.name,
		metrics: l.metrics,
	}
}

// GetName 获取名称
func (l *LambdaWithMiddleware[I, O]) GetName() string {
	return l.name
}

// GetMetrics 获取指标
func (l *LambdaWithMiddleware[I, O]) GetMetrics() LambdaMetrics {
	l.metrics.mu.RLock()
	defer l.metrics.mu.RUnlock()

	return LambdaMetrics{
		TotalInvocations:   l.metrics.TotalInvocations,
		SuccessInvocations: l.metrics.SuccessInvocations,
		ErrorInvocations:   l.metrics.ErrorInvocations,
		TotalDuration:      l.metrics.TotalDuration,
		AverageDuration:    l.metrics.AverageDuration,
		LastInvocationTime: l.metrics.LastInvocationTime,
	}
}

// ============================================================
// 内置中间件实现
// ============================================================

// Logger 日志中间件
func Logger[I any, O any](name string) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		start := time.Now()
		fmt.Printf("[%s] Started at %v\n", name, start.Format(time.RFC3339))

		// 调用下一个处理器
		output, err := next(ctx, input)

		duration := time.Since(start)
		if err != nil {
			fmt.Printf("[%s] Completed with error in %v: %v\n", name, duration, err)
		} else {
			fmt.Printf("[%s] Completed successfully in %v\n", name, duration)
		}

		return output, err
	}
}

// Recovery 恢复 panic 中间件
func Recovery[I any, O any]() Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (output O, err error) {
		defer func() {
			if r := recover(); r != nil {
				// 获取调用栈
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				err = fmt.Errorf("panic recovered: %v\nstack: %s", r, buf[:n])
				log.Printf("PANIC: %v\n%s", r, buf[:n])
			}
		}()

		return next(ctx, input)
	}
}

// Timeout 超时中间件
func Timeout[I any, O any](timeout time.Duration) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		resultChan := make(chan struct {
			output O
			err    error
		}, 1)

		go func() {
			output, err := next(ctx, input)
			select {
			case resultChan <- struct {
				output O
				err    error
			}{output, err}:
			case <-ctx.Done():
			}
		}()

		select {
		case res := <-resultChan:
			return res.output, res.err
		case <-ctx.Done():
			var zero O
			return zero, fmt.Errorf("timeout after %v", timeout)
		}
	}
}

// Retry 重试中间件
func Retry[I any, O any](maxRetries int) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		var lastErr error
		var zero O

		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				// 指数退避
				backoff := time.Duration(1<<uint(attempt-1)) * 100 * time.Millisecond
				if backoff > 5*time.Second {
					backoff = 5 * time.Second
				}

				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return zero, ctx.Err()
				}
			}

			output, err := next(ctx, input)
			if err == nil {
				return output, nil
			}

			lastErr = err

			// 如果是 context 错误，不重试
			if ctx.Err() != nil {
				return zero, ctx.Err()
			}
		}

		return zero, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
	}
}

// Metrics 指标收集中间件
func Metrics[I any, O any](metrics *LambdaMetrics) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		start := time.Now()

		output, err := next(ctx, input)

		duration := time.Since(start)

		// 更新指标
		metrics.mu.Lock()
		metrics.TotalInvocations++
		metrics.TotalDuration += duration
		metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.TotalInvocations)
		metrics.LastInvocationTime = time.Now()

		if err != nil {
			metrics.ErrorInvocations++
		} else {
			metrics.SuccessInvocations++
		}
		metrics.mu.Unlock()

		return output, err
	}
}

// ValidateInput 输入验证中间件
func ValidateInput[I any, O any](validator func(I) error) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		if err := validator(input); err != nil {
			var zero O
			return zero, fmt.Errorf("input validation failed: %w", err)
		}

		return next(ctx, input)
	}
}

// TransformInput 输入转换中间件
func TransformInput[I any, O any](transformer func(I) (I, error)) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		transformed, err := transformer(input)
		if err != nil {
			var zero O
			return zero, fmt.Errorf("input transformation failed: %w", err)
		}

		return next(ctx, transformed)
	}
}

// TransformOutput 输出转换中间件
func TransformOutput[I any, O any](transformer func(O) (O, error)) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		output, err := next(ctx, input)
		if err != nil {
			return output, err
		}

		transformed, err := transformer(output)
		if err != nil {
			var zero O
			return zero, fmt.Errorf("output transformation failed: %w", err)
		}

		return transformed, nil
	}
}

// CacheOutput 缓存输出中间件（简单实现）
func CacheOutput[I comparable, O any](cacheGetter func(I) (O, bool), cacheSetter func(I, O)) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		// 尝试从缓存获取
		if cached, found := cacheGetter(input); found {
			return cached, nil
		}

		// 调用下一个处理器
		output, err := next(ctx, input)
		if err != nil {
			return output, err
		}

		// 缓存结果
		cacheSetter(input, output)

		return output, nil
	}
}

// CircuitBreaker 熔断器中间件（简单实现）
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

type CircuitBreaker[I comparable] struct {
	maxFailures  int
	resetTimeout time.Duration
	lastFailure  time.Time
	state        CircuitBreakerState
	failures     map[I]int
}

func NewCircuitBreaker[I comparable](maxFailures int, resetTimeout time.Duration) *CircuitBreaker[I] {
	return &CircuitBreaker[I]{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
		failures:     make(map[I]int),
	}
}

func (cb *CircuitBreaker[I]) Middleware() Middleware[I, any] {
	return func(ctx context.Context, input I, next InvokeFunc[I, any]) (any, error) {
		// 检查熔断器状态
		if cb.state == CircuitOpen {
			if time.Since(cb.lastFailure) > cb.resetTimeout {
				cb.state = CircuitHalfOpen
			} else {
				return nil, fmt.Errorf("circuit breaker is OPEN for input: %v", input)
			}
		}

		output, err := next(ctx, input)

		// 记录失败
		if err != nil {
			cb.failures[input]++
			cb.lastFailure = time.Now()

			if cb.failures[input] >= cb.maxFailures {
				cb.state = CircuitOpen
			}

			return output, err
		}

		// 成功时重置
		if cb.state == CircuitHalfOpen {
			cb.state = CircuitClosed
		}
		cb.failures[input] = 0

		return output, nil
	}
}

// RateLimit 限流中间件（简单实现）
type RateLimiter struct {
	maxRequests int
	window      time.Duration
	requests    []time.Time
}

func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		maxRequests: maxRequests,
		window:      window,
		requests:    make([]time.Time, 0),
	}
}

func (rl *RateLimiter) Allow() bool {
	now := time.Now()

	// 清理过期的请求记录
	validIdx := 0
	for _, t := range rl.requests {
		if now.Sub(t) < rl.window {
			rl.requests[validIdx] = t
			validIdx++
		}
	}
	rl.requests = rl.requests[:validIdx]

	// 检查是否超过限制
	if len(rl.requests) >= rl.maxRequests {
		return false
	}

	rl.requests = append(rl.requests, now)
	return true
}

func RateLimit[I any, O any](limiter *RateLimiter) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		if !limiter.Allow() {
			var zero O
			return zero, fmt.Errorf("rate limit exceeded")
		}

		return next(ctx, input)
	}
}

// BeforeAfter 在处理器前后执行自定义逻辑
func BeforeAfter[I any, O any](
	before func(ctx context.Context, input I),
	after func(ctx context.Context, input I, output O, err error, duration time.Duration),
) Middleware[I, O] {
	return func(ctx context.Context, input I, next InvokeFunc[I, O]) (O, error) {
		start := time.Now()

		// 执行前置逻辑
		if before != nil {
			before(ctx, input)
		}

		// 调用下一个处理器
		output, err := next(ctx, input)
		duration := time.Since(start)

		// 执行后置逻辑
		if after != nil {
			after(ctx, input, output, err, duration)
		}

		return output, err
	}
}

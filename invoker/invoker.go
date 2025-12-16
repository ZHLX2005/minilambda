package invoker

import (
	"context"
	"fmt"
	"github.com/ZHLX2005/minilambda/core"
	"github.com/ZHLX2005/minilambda/registry"
	"sync"
	"time"
)

// Invoker lambda调用器
type Invoker[I any, O any] struct {
	semaphore chan struct{}
	mu        sync.RWMutex
}

// NewInvoker 创建新的调用器
func NewInvoker[I any, O any]() *Invoker[I, O] {
	return &Invoker[I, O]{} // 不使用注册表，简化实现
}

// Get 获取lambda (直接从全局注册表)
func (inv *Invoker[I, O]) Get(name string) (*core.Lambda[I, O], bool) {
	return registry.GetLambda[I, O](name)
}

// WithConcurrency 设置并发限制
func (inv *Invoker[I, O]) WithConcurrency(concurrency int) *Invoker[I, O] {
	inv.mu.Lock()
	defer inv.mu.Unlock()

	if concurrency > 0 {
		inv.semaphore = make(chan struct{}, concurrency)
	} else {
		inv.semaphore = nil
	}

	return inv
}

// Invoke 调用指定的lambda
func (inv *Invoker[I, O]) Invoke(ctx context.Context, name string, input I) (*core.LambdaResult[O], error) {
	// 获取lambda
	lambda, exists := inv.Get(name)
	if !exists {
		return nil, fmt.Errorf("lambda '%s' not found", name)
	}

	// 并发控制
	if inv.semaphore != nil {
		select {
		case inv.semaphore <- struct{}{}:
			defer func() { <-inv.semaphore }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 调用lambda
	return lambda.Invoke(ctx, input)
}

// InvokeAsync 异步调用lambda
func (inv *Invoker[I, O]) InvokeAsync(ctx context.Context, name string, input I) <-chan *core.LambdaResult[O] {
	resultChan := make(chan *core.LambdaResult[O], 1)

	go func() {
		defer close(resultChan)
		result, err := inv.Invoke(ctx, name, input)
		if err != nil {
			// 创建错误结果
			var zero O
			result = &core.LambdaResult[O]{
				Output:    zero,
				Error:     err,
				Duration:  0,
				Timestamp: time.Now(),
			}
		}
		resultChan <- result
	}()

	return resultChan
}

// InvokeMultiple 调用多个lambda
func (inv *Invoker[I, O]) InvokeMultiple(ctx context.Context, requests map[string]I) map[string]*core.LambdaResult[O] {
	results := make(map[string]*core.LambdaResult[O])
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, input := range requests {
		wg.Add(1)
		go func(nm string, inp I) {
			defer wg.Done()

			result, err := inv.Invoke(ctx, nm, inp)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				var zero O
				results[nm] = &core.LambdaResult[O]{
					Output:    zero,
					Error:     err,
					Duration:  0,
					Timestamp: time.Now(),
				}
			} else {
				results[nm] = result
			}
		}(name, input)
	}

	wg.Wait()
	return results
}

// Pipeline 管道式调用多个lambda
func (inv *Invoker[I, O]) Pipeline(ctx context.Context, name string, inputs []I) ([]*core.LambdaResult[O], error) {
	results := make([]*core.LambdaResult[O], len(inputs))

	for i, input := range inputs {
		result, err := inv.Invoke(ctx, name, input)
		if err != nil {
			return nil, fmt.Errorf("pipeline failed at step %d: %w", i, err)
		}
		results[i] = result

		// 如果有错误，停止管道
		if result.Error != nil {
			return results[:i+1], result.Error
		}
	}

	return results, nil
}

// Chain 链式调用多个不同的lambda，前一个的输出作为后一个的输入
func Chain[I any, O any](ctx context.Context, steps []ChainStep[I, O]) (*core.LambdaResult[O], error) {
	if len(steps) == 0 {
		return nil, fmt.Errorf("no steps in chain")
	}

	var currentInput interface{} = steps[0].Input
	var totalDuration time.Duration

	for i, step := range steps {
		// 类型断言
		typedInput, ok := currentInput.(I)
		if !ok {
			return nil, fmt.Errorf("type mismatch at step %d: expected %T, got %T", i, typedInput, currentInput)
		}

		inv := NewInvoker[I, O]()
		result, err := inv.Invoke(ctx, step.Name, typedInput)
		if err != nil {
			return nil, fmt.Errorf("chain failed at step %d (lambda: %s): %w", i, step.Name, err)
		}

		if result.Error != nil {
			return nil, fmt.Errorf("lambda failed at step %d (lambda: %s): %w", i, step.Name, result.Error)
		}

		totalDuration += result.Duration
		currentInput = result.Output

		// 如果是最后一步，返回结果
		if i == len(steps)-1 {
			finalOutput := currentInput.(O)
			return &core.LambdaResult[O]{
				Output:    finalOutput,
				Error:     nil,
				Duration:  totalDuration,
				Timestamp: time.Now(),
			}, nil
		}
	}

	return nil, fmt.Errorf("chain completed unexpectedly")
}

// ChainStep 链式调用步骤
type ChainStep[I any, O any] struct {
	Name  string
	Input I
}

// Retry 重试调用lambda
func (inv *Invoker[I, O]) Retry(ctx context.Context, name string, input I, maxRetries int, delay time.Duration) (*core.LambdaResult[O], error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := inv.Invoke(ctx, name, input)
		if err == nil && result.Error == nil {
			return result, nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = result.Error
		}
	}

	var zero O
	return &core.LambdaResult[O]{
		Output:    zero,
		Error:     fmt.Errorf("max retries exceeded, last error: %w", lastErr),
		Duration:  0,
		Timestamp: time.Now(),
	}, lastErr
}

// Timeout 带超时的调用
func (inv *Invoker[I, O]) Timeout(ctx context.Context, name string, input I, timeout time.Duration) (*core.LambdaResult[O], error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return inv.Invoke(ctx, name, input)
}

// Batch 批量调用同一个lambda
func (inv *Invoker[I, O]) Batch(ctx context.Context, name string, inputs []I, batchSize int) ([]*core.LambdaResult[O], error) {
	if batchSize <= 0 {
		batchSize = len(inputs)
	}

	var allResults []*core.LambdaResult[O]
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, 1)
	hasError := false

	for i := 0; i < len(inputs); i += batchSize {
		end := i + batchSize
		if end > len(inputs) {
			end = len(inputs)
		}

		batch := inputs[i:end]

		wg.Add(1)
		go func(batch []I, startIndex int) {
			defer wg.Done()

			batchResults := inv.InvokeMultiple(ctx, map[string]I{name: batch[0]}) // 简化处理

			mu.Lock()
			defer mu.Unlock()

			if hasError {
				return
			}

			for _, result := range batchResults {
				if result.Error != nil {
					hasError = true
					select {
					case errChan <- result.Error:
					default:
					}
					return
				}
			}

			// 添加结果到总结果
			allResults = append(allResults, batchResults[name])
		}(batch, i)
	}

	wg.Wait()

	select {
	case err := <-errChan:
		return allResults, err
	default:
		return allResults, nil
	}
}

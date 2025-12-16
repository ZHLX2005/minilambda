package core

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// NewLambda 创建新的lambda实例
func NewLambda[I any, O any](name string, invoke InvokeFunc[I, O], opts ...LambdaOption) *Lambda[I, O] {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	return &Lambda[I, O]{
		name:    name,
		invoke:  invoke,
		options: options,
		metrics: &LambdaMetrics{},
	}
}

// Invoke 调用lambda函数
func (l *Lambda[I, O]) Invoke(ctx context.Context, input I) (*LambdaResult[O], error) {
	start := time.Now()
	result := &LambdaResult[O]{
		Timestamp: start,
	}

	// 如果设置了超时，创建带超时的context
	if l.options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, l.options.Timeout)
		defer cancel()
	}

	// 执行lambda函数
	output, err := l.invokeWithRetry(ctx, input)

	result.Duration = time.Since(start)
	result.Output = output
	result.Error = err

	// 更新指标
	if l.options.EnableMetrics {
		l.updateMetrics(result.Duration, err)
	}

	return result, err
}

// invokeWithRetry 带重试的lambda调用
func (l *Lambda[I, O]) invokeWithRetry(ctx context.Context, input I) (O, error) {
	var lastErr error
	var zero O

	for attempt := 0; attempt <= l.options.Retries; attempt++ {
		if attempt > 0 {
			// 简单的重试延迟
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(time.Duration(attempt) * 100 * time.Millisecond):
			}
		}

		output, err := l.invoke(ctx, input)
		if err == nil {
			return output, nil
		}

		lastErr = err

		// 如果是context错误，不重试
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
	}

	return zero, lastErr
}

// updateMetrics 更新指标
func (l *Lambda[I, O]) updateMetrics(duration time.Duration, err error) {
	l.metrics.mu.Lock()
	defer l.metrics.mu.Unlock()

	l.metrics.TotalInvocations++
	l.metrics.TotalDuration += duration
	l.metrics.AverageDuration = l.metrics.TotalDuration / time.Duration(l.metrics.TotalInvocations)
	l.metrics.LastInvocationTime = time.Now()

	if err != nil {
		l.metrics.ErrorInvocations++
	} else {
		l.metrics.SuccessInvocations++
	}
}

// GetMetrics 获取指标
func (l *Lambda[I, O]) GetMetrics() LambdaMetrics {
	l.metrics.mu.RLock()
	defer l.metrics.mu.RUnlock()

	// 返回副本
	return LambdaMetrics{
		TotalInvocations:   l.metrics.TotalInvocations,
		SuccessInvocations: l.metrics.SuccessInvocations,
		ErrorInvocations:   l.metrics.ErrorInvocations,
		TotalDuration:      l.metrics.TotalDuration,
		AverageDuration:    l.metrics.AverageDuration,
		LastInvocationTime: l.metrics.LastInvocationTime,
	}
}

// GetName 获取lambda名称
func (l *Lambda[I, O]) GetName() string {
	return l.name
}

// GetOptions 获取配置选项
func (l *Lambda[I, O]) GetOptions() *LambdaOptions {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 返回选项的副本
	optsCopy := *l.options
	return &optsCopy
}

// GetMeta 获取lambda元数据
func (l *Lambda[I, O]) GetMeta() LambdaMeta {
	var inputType, outputType string

	// 使用反射获取类型信息
	inType := reflect.TypeOf((*I)(nil)).Elem()
	outType := reflect.TypeOf((*O)(nil)).Elem()

	inputType = inType.String()
	outputType = outType.String()

	return LambdaMeta{
		Name:          l.name,
		InputType:     inputType,
		OutputType:    outputType,
		ComponentType: l.options.ComponentType,
		RegisteredAt:  time.Now(),
	}
}

// WithOptions 创建带新选项的lambda副本
func (l *Lambda[I, O]) WithOptions(opts ...LambdaOption) *Lambda[I, O] {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 创建选项副本
	newOptions := *l.options
	for _, opt := range opts {
		opt(&newOptions)
	}

	return &Lambda[I, O]{
		name:    l.name,
		invoke:  l.invoke,
		options: &newOptions,
		metrics: l.metrics, // 共享指标
	}
}

// String 返回lambda的字符串表示
func (l *Lambda[I, O]) String() string {
	return fmt.Sprintf("Lambda[%s]: %s -> %s", l.name, l.GetMeta().InputType, l.GetMeta().OutputType)
}
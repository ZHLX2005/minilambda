package core

import (
	"context"
	"sync"
	"time"
)

// InvokeFunc 定义lambda调用函数类型
// I: 输入类型
// O: 输出类型
type InvokeFunc[I any, O any] func(ctx context.Context, input I) (output O, err error)

// InvokeFuncWithOptions 带选项的lambda调用函数类型
// I: 输入类型
// O: 输出类型
// TOption: 选项类型
type InvokeFuncWithOptions[I any, O any, TOption any] func(ctx context.Context, input I, opts ...TOption) (output O, err error)

// Lambda 核心lambda结构体
type Lambda[I any, O any] struct {
	name      string
	invoke    InvokeFunc[I, O]
	options   *LambdaOptions
	mu        sync.RWMutex
	metrics   *LambdaMetrics
}

// LambdaOptions lambda配置选项
type LambdaOptions struct {
	// 超时时间
	Timeout time.Duration
	// 是否启用指标收集
	EnableMetrics bool
	// 并发限制
	Concurrency int
	// 重试次数
	Retries int
	// 是否启用组件回调
	EnableCallback bool
	// 组件实现类型
	ComponentType string
}

// LambdaMetrics lambda指标统计
type LambdaMetrics struct {
	mu                sync.RWMutex
	TotalInvocations   int64
	SuccessInvocations int64
	ErrorInvocations   int64
	TotalDuration      time.Duration
	AverageDuration    time.Duration
	LastInvocationTime time.Time
}

// LambdaResult lambda调用结果
type LambdaResult[O any] struct {
	Output    O
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}

// LambdaMeta lambda元数据
type LambdaMeta struct {
	Name          string
	InputType     string
	OutputType    string
	ComponentType string
	RegisteredAt  time.Time
}

// 默认选项
func DefaultOptions() *LambdaOptions {
	return &LambdaOptions{
		Timeout:        30 * time.Second,
		EnableMetrics:  true,
		Concurrency:    10,
		Retries:        0,
		EnableCallback: false,
		ComponentType:  "Lambda",
	}
}

// LambdaOption lambda选项函数
type LambdaOption func(*LambdaOptions)

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) LambdaOption {
	return func(opts *LambdaOptions) {
		opts.Timeout = timeout
	}
}

// WithEnableMetrics 设置是否启用指标收集
func WithEnableMetrics(enable bool) LambdaOption {
	return func(opts *LambdaOptions) {
		opts.EnableMetrics = enable
	}
}

// WithConcurrency 设置并发限制
func WithConcurrency(concurrency int) LambdaOption {
	return func(opts *LambdaOptions) {
		opts.Concurrency = concurrency
	}
}

// WithRetries 设置重试次数
func WithRetries(retries int) LambdaOption {
	return func(opts *LambdaOptions) {
		opts.Retries = retries
	}
}

// WithEnableCallback 设置是否启用回调
func WithEnableCallback(enable bool) LambdaOption {
	return func(opts *LambdaOptions) {
		opts.EnableCallback = enable
	}
}

// WithComponentType 设置组件类型
func WithComponentType(componentType string) LambdaOption {
	return func(opts *LambdaOptions) {
		opts.ComponentType = componentType
	}
}
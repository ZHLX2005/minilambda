package registry

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/ZHLX2005/minilambda/core"
)

// AutoRegisterer 自动注册器
type AutoRegisterer struct {
	mu       sync.RWMutex
	handlers []func()
}

var globalAutoRegisterer = &AutoRegisterer{}

// RegisterHandler 注册自动处理函数
func (ar *AutoRegisterer) RegisterHandler(handler func()) {
	ar.mu.Lock()
	defer ar.mu.Unlock()
	ar.handlers = append(ar.handlers, handler)
}

// ExecuteHandlers 执行所有处理函数
func (ar *AutoRegisterer) ExecuteHandlers() {
	ar.mu.RLock()
	handlers := make([]func(), len(ar.handlers))
	copy(handlers, ar.handlers)
	ar.mu.RUnlock()

	for _, handler := range handlers {
		handler()
	}
}

// RegisterAutoHandler 注册自动处理函数到全局注册器
func RegisterAutoHandler(handler func()) {
	globalAutoRegisterer.RegisterHandler(handler)
}

// ExecuteAutoHandlers 执行所有自动处理函数
func ExecuteAutoHandlers() {
	globalAutoRegisterer.ExecuteHandlers()
}

// LambdaRegisterer lambda注册器接口
type LambdaRegisterer interface {
	RegisterLambdas() error
}

// RegisterAutoLambdas 自动注册lambda
func RegisterAutoLambdas(registerer LambdaRegisterer) error {
	return registerer.RegisterLambdas()
}

// ScanPackage 扫描包并自动注册lambda函数
// 这个函数使用反射来查找符合条件的函数
func ScanPackage(packageName string) error {
	// 注意：在Go中，运行时扫描包需要使用go/parser和go/ast
	// 这里提供一个简化的框架

	// 实际实现需要：
	// 1. 解析Go源文件
	// 2. 查找符合lambda函数签名的方法
	// 3. 自动注册这些函数

	return fmt.Errorf("package scanning not yet implemented: %s", packageName)
}

// RegisterByFunction 通过函数注册lambda
// 函数签名必须符合: func(ctx context.Context, input I) (O, error)
func RegisterByFunction[I any, O any](name string, fn interface{}, opts ...core.LambdaOption) error {
	// 检查函数类型
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("expected function, got %T", fn)
	}

	// 检查函数签名
	if fnType.NumIn() != 2 || fnType.NumOut() != 2 {
		return fmt.Errorf("function must have signature: func(context.Context, I) (O, error)")
	}

	// 检查参数类型
	contextType := reflect.TypeOf((*interface{})(nil)).Elem()
	inputType := reflect.TypeOf((*I)(nil)).Elem()

	if !fnType.In(0).Implements(contextType) {
		return fmt.Errorf("first parameter must be context.Context")
	}

	if fnType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return fmt.Errorf("first parameter must be context.Context")
	}

	if fnType.In(1) != inputType {
		return fmt.Errorf("second parameter must be of type %s", inputType)
	}

	// 检查返回类型
	outputType := reflect.TypeOf((*O)(nil)).Elem()
	errorType := reflect.TypeOf((*error)(nil)).Elem()

	if fnType.Out(0) != outputType {
		return fmt.Errorf("first return value must be of type %s", outputType)
	}

	if !fnType.Out(1).Implements(errorType) {
		return fmt.Errorf("second return value must implement error")
	}

	// 创建lambda函数
	invoke := func(ctx context.Context, input I) (O, error) {
		args := []reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(input),
		}

		results := reflect.ValueOf(fn).Call(args)
		output := results[0].Interface().(O)
		err := results[1].Interface().(error)

		return output, err
	}

	// 注册lambda
	return RegisterLambda(name, invoke, opts...)
}

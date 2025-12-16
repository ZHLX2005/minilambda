package registry

import (
	"fmt"
	"reflect"
	"sync"

	"minilambda/core"
)

// GlobalRegistry 全局注册中心
var GlobalRegistry = NewRegistry()

// Registry 泛型lambda注册中心
type Registry[I any, O any] struct {
	mu           sync.RWMutex
	lambdas      map[string]*core.Lambda[I, O]
	constructors map[string]func() *core.Lambda[I, O]
	meta         map[string]core.LambdaMeta
}

// globalRegistries 存储所有泛型类型组合的注册表
var globalRegistries = sync.Map{}

// NewRegistry 创建新的注册中心
func NewRegistry() *Registry[string, string] {
	return &Registry[string, string]{
		lambdas:      make(map[string]*core.Lambda[string, string]),
		constructors: make(map[string]func() *core.Lambda[string, string]),
		meta:         make(map[string]core.LambdaMeta),
	}
}

// getRegistry 获取或创建指定泛型类型的注册表
func getRegistry[I any, O any]() *Registry[I, O] {
	key := registryKey[I, O]()

	if reg, ok := globalRegistries.Load(key); ok {
		return reg.(*Registry[I, O])
	}

	reg := &Registry[I, O]{
		lambdas:      make(map[string]*core.Lambda[I, O]),
		constructors: make(map[string]func() *core.Lambda[I, O]),
		meta:         make(map[string]core.LambdaMeta),
	}

	globalRegistries.Store(key, reg)
	return reg
}

// registryKey 生成泛型类型的唯一键
func registryKey[I any, O any]() string {
	inType := reflect.TypeOf((*I)(nil)).Elem()
	outType := reflect.TypeOf((*O)(nil)).Elem()
	return inType.String() + "->" + outType.String()
}

// Register 注册lambda
func (r *Registry[I, O]) Register(lambda *core.Lambda[I, O]) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := lambda.GetName()
	if _, exists := r.lambdas[name]; exists {
		return fmt.Errorf("lambda '%s' already registered", name)
	}

	r.lambdas[name] = lambda
	r.meta[name] = lambda.GetMeta()
	return nil
}

// RegisterWithConstructor 注册lambda构造函数
func (r *Registry[I, O]) RegisterWithConstructor(name string, constructor func() *core.Lambda[I, O]) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.constructors[name] = constructor
}

// Get 获取lambda
func (r *Registry[I, O]) Get(name string) (*core.Lambda[I, O], bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	lambda, exists := r.lambdas[name]
	return lambda, exists
}

// Build 使用构造函数创建lambda
func (r *Registry[I, O]) Build(name string) (*core.Lambda[I, O], error) {
	r.mu.RLock()
	constructor, exists := r.constructors[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("constructor for lambda '%s' not found", name)
	}

	lambda := constructor()

	// 注册创建的lambda
	r.Register(lambda)

	return lambda, nil
}

// List 列出所有注册的lambda名称
func (r *Registry[I, O]) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.lambdas)+len(r.constructors))

	// 添加已注册的lambda名称
	for name := range r.lambdas {
		names = append(names, name)
	}

	// 添加构造函数名称（如果还没有对应的lambda）
	for name := range r.constructors {
		if _, exists := r.lambdas[name]; !exists {
			names = append(names, name)
		}
	}

	return names
}

// GetMeta 获取lambda元数据
func (r *Registry[I, O]) GetMeta(name string) (core.LambdaMeta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	meta, exists := r.meta[name]
	return meta, exists
}

// GetAllMeta 获取所有lambda元数据
func (r *Registry[I, O]) GetAllMeta() map[string]core.LambdaMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metaCopy := make(map[string]core.LambdaMeta)
	for name, meta := range r.meta {
		metaCopy[name] = meta
	}

	return metaCopy
}

// Unregister 注销lambda
func (r *Registry[I, O]) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.lambdas[name]; exists {
		delete(r.lambdas, name)
		delete(r.meta, name)
		return true
	}

	if _, exists := r.constructors[name]; exists {
		delete(r.constructors, name)
		return true
	}

	return false
}

// Clear 清空注册表
func (r *Registry[I, O]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lambdas = make(map[string]*core.Lambda[I, O])
	r.constructors = make(map[string]func() *core.Lambda[I, O])
	r.meta = make(map[string]core.LambdaMeta)
}

// Count 返回注册的lambda数量
func (r *Registry[I, O]) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.lambdas) + len(r.constructors)
}

// 全局注册函数

// RegisterLambda 注册lambda到全局注册表
func RegisterLambda[I any, O any](name string, invoke core.InvokeFunc[I, O], opts ...core.LambdaOption) error {
	lambda := core.NewLambda(name, invoke, opts...)
	reg := getRegistry[I, O]()
	return reg.Register(lambda)
}

// RegisterLambdaWithConstructor 注册lambda构造函数到全局注册表
func RegisterLambdaWithConstructor[I any, O any](name string, constructor func() *core.Lambda[I, O]) {
	reg := getRegistry[I, O]()
	reg.RegisterWithConstructor(name, constructor)
}

// GetLambda 从全局注册表获取lambda
func GetLambda[I any, O any](name string) (*core.Lambda[I, O], bool) {
	reg := getRegistry[I, O]()
	return reg.Get(name)
}

// BuildLambda 从全局注册表构建lambda
func BuildLambda[I any, O any](name string) (*core.Lambda[I, O], error) {
	reg := getRegistry[I, O]()
	return reg.Build(name)
}

// ListLambdas 列出指定泛型类型的所有lambda
func ListLambdas[I any, O any]() []string {
	reg := getRegistry[I, O]()
	return reg.List()
}

// GetLambdaMeta 从全局注册表获取lambda元数据
func GetLambdaMeta[I any, O any](name string) (core.LambdaMeta, bool) {
	reg := getRegistry[I, O]()
	return reg.GetMeta(name)
}

// UnregisterLambda 从全局注册表注销lambda
func UnregisterLambda[I any, O any](name string) bool {
	reg := getRegistry[I, O]()
	return reg.Unregister(name)
}

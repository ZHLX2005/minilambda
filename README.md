# MiniLambda

一个轻量级的Go语言Lambda函数框架，灵感来源于Eino的Lambda设计，专注于同步调用场景。

## 特性

- **泛型支持**: 完全支持Go泛型，类型安全
- **注册中心**: 灵活的lambda函数注册和查找机制
- **自动注册**: 支持包级别的自动lambda注册
- **调用器**: 多种调用模式（同步、异步、批量、管道）
- **指标监控**: 内置调用指标收集
- **选项配置**: 丰富的配置选项（超时、重试、并发控制等）
- **🆕 中间件系统**: 类似Gin的责任链模式，支持灵活的中间件组合

## 快速开始

### 1. 基本用法

```go
package main

import (
    "context"
    "fmt"
    "github.com/minilambda/core"
    "github.com/minilambda/registry"
)

func main() {
    // 注册lambda函数
    registry.RegisterLambda("string_upper", func(ctx context.Context, input string) (string, error) {
        return strings.ToUpper(input), nil
    })

    // 创建调用器并调用
    inv := invoker.NewInvoker[string, string]()
    result, err := inv.Invoke(context.Background(), "string_upper", "hello")
    if err != nil {
        panic(err)
    }

    fmt.Println(result.Output) // "HELLO"
}
```

### 2. 带选项的Lambda

```go
// 创建带配置的lambda
lambda := core.NewLambda("my_lambda",
    func(ctx context.Context, input int) (int, error) {
        return input * 2, nil
    },
    core.WithTimeout(5*time.Second),
    core.WithEnableMetrics(true),
    core.WithRetries(3),
)

// 调用
result, err := lambda.Invoke(context.Background(), 21)
fmt.Println(result.Output) // 42
```

### 3. 自动注册

```go
// 在包的init函数中注册lambda
func init() {
    registry.RegisterAutoHandler(registerMyLambdas)
}

func registerMyLambdas() {
    registry.RegisterLambda("process_data", processData)
    registry.RegisterLambda("validate_input", validateInput)
}

// 在主程序中初始化
func main() {
    minilambda.Init() // 执行所有自动注册
}
```

### 4. 🆕 中间件系统

```go
// 创建带中间件的 Lambda（类似 Gin 的责任链）
lambda := core.NewLambdaWithMiddleware(
    "order_processor",
    processOrder,  // 处理函数
    core.Logger[Request, Response]("OrderProcessor"),
    core.Recovery[Request, Response](),
    core.Timeout[Request, Response](30*time.Second),
    core.Retry[Request, Response](3),
)

// 调用
result, err := lambda.Invoke(ctx, orderRequest)

// 动态添加中间件
lambdaWithAuth := lambda.Use(
    AuthMiddleware("admin"),
    core.RateLimit[Request, Response](limiter),
)
```

详细的中间件文档请查看 [MIDDLEWARE.md](MIDDLEWARE.md)

## 核心组件

### 1. Lambda类型

```go
// 基本lambda函数类型
type InvokeFunc[I any, O any] func(ctx context.Context, input I) (output O, err error)

// Lambda结构体
type Lambda[I any, O any] struct {
    name      string
    invoke    InvokeFunc[I, O]
    options   *LambdaOptions
    metrics   *LambdaMetrics
}
```

### 2. 注册中心

```go
// 注册lambda
err := registry.RegisterLambda("name", lambdaFunc)

// 获取lambda
lambda, exists := registry.GetLambda[int, string]("name")

// 列出所有lambda
names := registry.ListLambdas[int, string]()
```

### 3. 调用器

```go
inv := invoker.NewInvoker[int, string]()

// 同步调用
result, err := inv.Invoke(ctx, "lambda_name", 42)

// 异步调用
resultChan := inv.InvokeAsync(ctx, "lambda_name", 42)

// 批量调用
requests := map[string]int{"lambda1": 1, "lambda2": 2}
results := inv.InvokeMultiple(ctx, requests)

// 管道调用
inputs := []int{1, 2, 3, 4, 5}
results, err := inv.Pipeline(ctx, "lambda_name", inputs)
```

## 配置选项

### LambdaOptions

```go
type LambdaOptions struct {
    Timeout        time.Duration  // 超时时间
    EnableMetrics  bool           // 启用指标收集
    Concurrency    int            // 并发限制
    Retries        int            // 重试次数
    EnableCallback bool           // 启用组件回调
    ComponentType  string         // 组件类型
}
```

### 可用选项

- `WithTimeout(time.Duration)` - 设置超时时间
- `WithEnableMetrics(bool)` - 启用/禁用指标收集
- `WithConcurrency(int)` - 设置并发限制
- `WithRetries(int)` - 设置重试次数
- `WithEnableCallback(bool)` - 启用/禁用组件回调
- `WithComponentType(string)` - 设置组件类型

## 指标监控

```go
// 获取lambda指标
metrics := lambda.GetMetrics()
fmt.Printf("Total invocations: %d\n", metrics.TotalInvocations)
fmt.Printf("Success rate: %.2f%%\n",
    float64(metrics.SuccessInvocations)/float64(metrics.TotalInvocations)*100)
fmt.Printf("Average duration: %v\n", metrics.AverageDuration)
```

## 项目结构

```
minilambda/
├── core/              # 核心类型定义
│   ├── types.go       # Lambda核心类型
│   ├── lambda.go      # Lambda实现
│   └── middleware.go  # 🆕 中间件系统
├── registry/          # 注册中心
│   ├── registry.go    # 注册中心实现
│   └── auto_register.go # 自动注册
├── invoker/           # 调用器
│   └── invoker.go     # 调用器实现
├── example/           # 示例代码
│   ├── lambdas.go     # 示例lambda函数
│   ├── demo.go        # 演示程序
│   └── middleware_demo.go # 🆕 中间件演示
├── test/             # 测试代码
│   └── lambda_test.go
├── init.go        # 包初始化
├── go.mod
└── README.md
```

## 运行示例

```bash
# 运行基本演示程序
go run minilambda/example/demo.go

# 运行中间件演示程序（推荐）
go run minilambda/example/middleware_demo.go

# 运行测试
go test ./minilambda/test/...

# 运行基准测试
go test -bench=. ./minilambda/test/...
```

## 与Eino的对比

MiniLambda专注于同步调用场景，相比Eino：

- **简化设计**: 移除了stream相关的复杂逻辑
- **更轻量**: 核心代码更少，启动更快
- **类型安全**: 完全基于Go泛型实现
- **易于集成**: 更简单的API设计

## 许可证

Apache License 2.0

## 贡献

欢迎提交Issue和Pull Request！
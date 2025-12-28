# MiniLambda 中间件系统

一套类似 Gin 框架的中间件系统，支持责任链模式，可以灵活地组合和复用业务逻辑。

## 核心概念

### 中间件类型

```go
type Middleware[I any, O any] func(
    ctx context.Context,
    input I,
    next InvokeFunc[I, O],
) (O, error)
```

中间件函数接收：
- `ctx`: 上下文
- `input`: 输入数据
- `next`: 下一个处理函数（调用它来传递控制权）

### 责任链执行流程

```
请求 → 中间件1 → 中间件2 → ... → 处理器
        ↓         ↓
      前置逻辑  前置逻辑
        ↑         ↑
      后置逻辑  后置逻辑
```

## 快速开始

### 1. 基本用法

```go
package main

import (
    "context"
    "fmt"
    "github.com/ZHLX2005/minilambda/core"
)

// 业务处理函数
func processOrder(ctx context.Context, orderID string) (string, error) {
    // 处理订单逻辑
    return fmt.Sprintf("Order %s processed", orderID), nil
}

func main() {
    // 创建带中间件的 Lambda
    lambda := core.NewLambdaWithMiddleware(
        "order_processor",
        processOrder,
        core.Logger[string, string]("OrderProcessor"),
        core.Recovery[string, string](),
        core.Timeout[string, string](30*time.Second),
    )

    // 调用
    result, err := lambda.Invoke(context.Background(), "ORD-12345")
    if err != nil {
        panic(err)
    }

    fmt.Println(result.Output)
}
```

### 2. 动态添加中间件

```go
// 创建基础 Lambda
baseLambda := core.NewLambdaWithMiddleware(
    "service",
    handler,
    core.Logger[int, string]("Service"),
)

// 动态添加更多中间件
enhancedLambda := baseLambda.Use(
    core.Timeout[int, string](5*time.Second),
    core.Retry[int, string](3),
)
```

### 3. 自定义中间件

```go
// 认证中间件
func AuthMiddleware[I any, O any](requiredRole string) core.Middleware[I, O] {
    return func(ctx context.Context, input I, next core.InvokeFunc[I, O]) (O, error) {
        // 检查认证
        token := ctx.Value("token")
        if token == nil {
            var zero O
            return zero, errors.New("unauthorized")
        }

        // 调用下一个处理器
        return next(ctx, input)
    }
}

// 使用
lambda := core.NewLambdaWithMiddleware(
    "protected_resource",
    handler,
    AuthMiddleware[int, string]("admin"),
)
```

## 内置中间件

### Logger - 日志中间件

记录请求的开始、结束时间和状态。

```go
lambda := core.NewLambdaWithMiddleware(
    "my_service",
    handler,
    core.Logger[Input, Output]("MyService"),
)
```

### Recovery - 恢复中间件

捕获 panic 并转换为错误。

```go
lambda := core.NewLambdaWithMiddleware(
    "my_service",
    handler,
    core.Recovery[Input, Output](),
)
```

### Timeout - 超时中间件

设置处理超时时间。

```go
lambda := core.NewLambdaWithMiddleware(
    "my_service",
    handler,
    core.Timeout[Input, Output](30*time.Second),
)
```

### Retry - 重试中间件

失败时自动重试，支持指数退避。

```go
lambda := core.NewLambdaWithMiddleware(
    "my_service",
    handler,
    core.Retry[Input, Output](3), // 最多重试3次
)
```

### Metrics - 指标中间件

收集调用统计信息。

```go
metrics := &core.LambdaMetrics{}

lambda := core.NewLambdaWithMiddleware(
    "my_service",
    handler,
    core.Metrics[Input, Output](metrics),
)

// 获取指标
stats := lambda.GetMetrics()
fmt.Printf("Total: %d, Success: %d, Errors: %d\n",
    stats.TotalInvocations,
    stats.SuccessInvocations,
    stats.ErrorInvocations,
)
```

### ValidateInput - 输入验证中间件

```go
validate := core.ValidateInput[Request, Response](func(req Request) error {
    if req.Username == "" {
        return errors.New("username is required")
    }
    return nil
})

lambda := core.NewLambdaWithMiddleware(
    "login",
    loginHandler,
    validate,
)
```

### TransformInput/TransformOutput - 数据转换中间件

```go
// 输入转换
sanitizeInput := core.TransformInput[Request, Response](func(req Request) (Request, error) {
    req.Username = strings.TrimSpace(req.Username)
    return req, nil
})

// 输出转换
encryptOutput := core.TransformOutput[Request, Response](func(resp Response) (Response, error) {
    resp.Token = encrypt(resp.Token)
    return resp, nil
})
```

### CacheOutput - 缓存中间件

```go
cache := make(map[string]Result)

cacheMiddleware := core.CacheOutput(
    func(key string) (Result, bool) { // getter
        val, ok := cache[key]
        return val, ok
    },
    func(key string, val Result) { // setter
        cache[key] = val
    },
)
```

### RateLimit - 限流中间件

```go
limiter := core.NewRateLimiter(100, time.Second) // 每秒100个请求

lambda := core.NewLambdaWithMiddleware(
    "api",
    handler,
    core.RateLimit[Request, Response](limiter),
)
```

### BeforeAfter - 前后置逻辑中间件

```go
middleware := core.BeforeAfter[Request, Response](
    func(ctx context.Context, req Request) {
        log.Printf("Before: %v", req)
    },
    func(ctx context.Context, req Request, resp Response, err error, duration time.Duration) {
        log.Printf("After: duration=%v, err=%v", duration, err)
    },
)
```

## 高级用法

### 1. 中间件链组合

```go
// 基础中间件
baseMiddlewares := []core.Middleware[Request, Response]{
    core.Logger[Request, Response]("Service"),
    core.Recovery[Request, Response](),
}

// 特定中间件
authMiddlewares := []core.Middleware[Request, Response]{
    AuthMiddleware("admin"),
    AuditLog(),
}

// 组合使用
lambda := core.NewLambdaWithMiddleware(
    "admin_service",
    adminHandler,
)
lambda = lambda.Use(baseMiddlewares...).Use(authMiddlewares...)
```

### 2. 条件中间件

```go
func ConditionalMiddleware[I any, O any](
    condition func(I) bool,
    middleware core.Middleware[I, O],
) core.Middleware[I, O] {
    return func(ctx context.Context, input I, next core.InvokeFunc[I, O]) (O, error) {
        if condition(input) {
            return middleware(ctx, input, next)
        }
        return next(ctx, input)
    }
}

// 使用
lambda := core.NewLambdaWithMiddleware(
    "service",
    handler,
    ConditionalMiddleware(
        func(req Request) bool { return req.RequireAuth },
        AuthMiddleware("user"),
    ),
)
```

### 3. 中间件间共享状态

```go
// 使用 context 传递状态
func StateMiddleware[I any, O any](key interface{}, value interface{}) core.Middleware[I, O] {
    return func(ctx context.Context, input I, next core.InvokeFunc[I, O]) (O, error) {
        ctx = context.WithValue(ctx, key, value)
        return next(ctx, input)
    }
}

// 在后续中间件中使用
func GetState[I any, O any](key interface{}) core.Middleware[I, O] {
    return func(ctx context.Context, input I, next core.InvokeFunc[I, O]) (O, error) {
        value := ctx.Value(key)
        // 使用 value
        return next(ctx, input)
    }
}
```

### 4. 自定义熔断器中间件

```go
type CircuitBreaker struct {
    failures    int
    maxFailures int
    openUntil   time.Time
    resetTimeout time.Duration
}

func (cb *CircuitBreaker) Middleware[I any, O any]() core.Middleware[I, O] {
    return func(ctx context.Context, input I, next core.InvokeFunc[I, O]) (O, error) {
        if time.Now().Before(cb.openUntil) {
            var zero O
            return zero, errors.New("circuit breaker is OPEN")
        }

        output, err := next(ctx, input)

        if err != nil {
            cb.failures++
            if cb.failures >= cb.maxFailures {
                cb.openUntil = time.Now().Add(cb.resetTimeout)
            }
            return output, err
        }

        cb.failures = 0
        return output, nil
    }
}
```

## 最佳实践

### 1. 中间件顺序建议

推荐的中间件顺序：

```go
lambda := core.NewLambdaWithMiddleware(
    "service",
    handler,
    // 1. 最外层：日志和恢复
    core.Logger[Input, Output]("Service"),
    core.Recovery[Input, Output](),

    // 2. 限流和熔断
    core.RateLimit[Input, Output](limiter),
    circuitBreaker.Middleware(),

    // 3. 认证和授权
    AuthMiddleware("user"),

    // 4. 验证和转换
    core.ValidateInput(validator),
    core.TransformInput(sanitizer),

    // 5. 重试和超时
    core.Retry[Input, Output](3),
    core.Timeout[Input, Output](30*time.Second),

    // 6. 缓存
    core.CacheOutput(getter, setter),

    // 7. 指标收集
    core.Metrics[Input, Output](metrics),
)
```

### 2. 中间件设计原则

- **单一职责**：每个中间件只做一件事
- **可组合性**：中间件应该可以任意组合
- **幂等性**：多次调用应该产生相同的结果
- **无状态性**：避免在中间件中存储可变状态
- **错误处理**：明确处理错误情况，决定是否继续传递

### 3. 性能考虑

```go
// 使用池化减少分配
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 1024)
    },
}

func BufferMiddleware[I any, O any]() core.Middleware[I, O] {
    return func(ctx context.Context, input I, next core.InvokeFunc[I, O]) (O, error) {
        buf := bufferPool.Get().([]byte)
        defer func() {
            bufferPool.Put(buf[:0])
        }()

        // 使用 buf...

        return next(ctx, input)
    }
}
```

## 运行示例

```bash
# 运行中间件演示
cd minilambda
go run example/middleware_demo.go

# 运行测试
go test ./core/...

# 运行基准测试
go test -bench=. ./core/...
```

## 与 Gin 的对比

| 特性 | Gin (Web) | MiniLambda (通用) |
|------|-----------|------------------|
| 类型 | HTTP 专用 | 任意类型 |
| 上下文 | *gin.Context | context.Context |
| 函数签名 | HandlerFunc | InvokeFunc[I, O] |
| 泛型支持 | ❌ | ✅ |
| 中间件类型 | HandlerFunc | Middleware[I, O] |
| Next() 调用 | c.Next() | next(ctx, input) |

## 总结

MiniLambda 的中间件系统提供了：

1. **类型安全**：完全的泛型支持，编译时类型检查
2. **灵活性**：可以组合任意数量的中间件
3. **可复用性**：中间件可以在不同的 Lambda 之间共享
4. **可测试性**：每个中间件都可以独立测试
5. **可扩展性**：轻松创建自定义中间件

这套系统不仅限于 HTTP 处理，可以用于任何需要责任链模式的场景。

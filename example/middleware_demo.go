package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ZHLX2005/minilambda/core"
)

// ============================================================
// 业务模型定义
// ============================================================

// User 用户结构
type User struct {
	ID    int
	Name  string
	Email string
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string
	Password string
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token    string
	UserID   int
	ExpireAt time.Time
}

// ============================================================
// 业务处理函数
// ============================================================

// SimpleBusinessHandler 简单的业务处理函数
func SimpleBusinessHandler(ctx context.Context, input string) (string, error) {
	fmt.Printf("  [Handler] Processing: %s\n", input)
	time.Sleep(100 * time.Millisecond)
	return fmt.Sprintf("PROCESSED: %s", input), nil
}

// LoginHandler 登录处理函数
func LoginHandler(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	fmt.Printf("  [Handler] Authenticating user: %s\n", req.Username)

	// 模拟数据库查询
	time.Sleep(50 * time.Millisecond)

	if req.Username == "admin" && req.Password == "secret" {
		return LoginResponse{
			Token:    "jwt_token_123456",
			UserID:   1,
			ExpireAt: time.Now().Add(24 * time.Hour),
		}, nil
	}

	return LoginResponse{}, errors.New("invalid username or password")
}

// GetUserHandler 获取用户处理函数
func GetUserHandler(ctx context.Context, userID int) (User, error) {
	fmt.Printf("  [Handler] Fetching user with ID: %d\n", userID)

	// 模拟数据库查询
	time.Sleep(30 * time.Millisecond)

	return User{
		ID:    userID,
		Name:  "John Doe",
		Email: "john@example.com",
	}, nil
}

// FailingHandler 会失败的处理器（用于测试重试和熔断）
func FailingHandler(ctx context.Context, input int) (string, error) {
	fmt.Printf("  [Handler] Attempting to process: %d\n", input)
	time.Sleep(10 * time.Millisecond)

	// 前两次调用失败
	if input < 3 {
		return "", fmt.Errorf("temporary error for input: %d", input)
	}

	return fmt.Sprintf("Success: %d", input), nil
}

// ============================================================
// 自定义中间件
// ============================================================

// Auth 认证中间件
func Auth[I any, O any](requiredRole string) core.Middleware[I, O] {
	return func(ctx context.Context, input I, next core.InvokeFunc[I, O]) (O, error) {
		fmt.Printf("  [Auth] Checking required role: %s\n", requiredRole)

		// 从 context 获取用户信息
		if userID := ctx.Value("user_id"); userID == nil {
			var zero O
			return zero, errors.New("unauthorized: no user in context")
		}

		fmt.Printf("  [Auth] User authenticated\n")
		return next(ctx, input)
	}
}

// AuditLog 审计日志中间件
func AuditLog[I any, O any]() core.Middleware[I, O] {
	return func(ctx context.Context, input I, next core.InvokeFunc[I, O]) (O, error) {
		fmt.Printf("  [Audit] Request received at %v\n", time.Now().Format(time.RFC3339))

		output, err := next(ctx, input)

		status := "SUCCESS"
		if err != nil {
			status = "FAILED"
		}
		fmt.Printf("  [Audit] Request completed: %s\n", status)

		return output, err
	}
}

// SanitizeInput 输入清理中间件
func SanitizeInput() core.Middleware[string, string] {
	return func(ctx context.Context, input string, next core.InvokeFunc[string, string]) (string, error) {
		// 简单的输入清理
		cleaned := input
		if len(input) > 100 {
			cleaned = input[:100] + "..."
			fmt.Printf("  [Sanitize] Input truncated from %d to 103 chars\n", len(input))
		}

		return next(ctx, cleaned)
	}
}

// ============================================================
// 主函数
// ============================================================

func main() {
	fmt.Println("========================================")
	fmt.Println("MiniLambda Middleware Chain Demo")
	fmt.Println("========================================\n")

	// Demo 1: 基础中间件链
	fmt.Println("1. Basic Middleware Chain:")
	demoBasicChain()

	// Demo 2: 登录处理链
	fmt.Println("\n\n2. Login Processing Chain:")
	demoLoginChain()

	// Demo 3: 动态添加中间件
	fmt.Println("\n\n3. Dynamic Middleware Addition:")
	demoDynamicMiddleware()

	// Demo 4: 重试机制
	fmt.Println("\n\n4. Retry Mechanism:")
	demoRetry()

	// Demo 5: 限流
	fmt.Println("\n\n5. Rate Limiting:")
	demoRateLimit()

	// Demo 6: 熔断器
	fmt.Println("\n\n6. Circuit Breaker:")
	demoCircuitBreaker()

	// Demo 7: 自定义中间件
	fmt.Println("\n\n7. Custom Middleware:")
	demoCustomMiddleware()

	fmt.Println("\n========================================")
	fmt.Println("All demos completed!")
	fmt.Println("========================================")
}

// demoBasicChain 基础中间件链演示
func demoBasicChain() {
	// 创建带中间件的 Lambda
	lambda := core.NewLambdaWithMiddleware(
		"basic_processor",
		SimpleBusinessHandler,
		// 中间件按顺序执行
		core.Logger[string, string]("BasicProcessor"),
		core.Recovery[string, string](),
		core.Timeout[string, string](200*time.Millisecond),
	)

	// 调用
	ctx := context.Background()
	result, err := lambda.Invoke(ctx, "Hello, Middleware Chain!")

	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	fmt.Printf("  Result: %s\n", result.Output)
	fmt.Printf("  Duration: %v\n", result.Duration)
}

// demoLoginChain 登录处理链演示
func demoLoginChain() {
	// 输入验证中间件
	validateLogin := core.ValidateInput[LoginRequest, LoginResponse](func(req LoginRequest) error {
		if req.Username == "" {
			return errors.New("username is required")
		}
		if req.Password == "" {
			return errors.New("password is required")
		}
		if len(req.Password) < 6 {
			return errors.New("password must be at least 6 characters")
		}
		return nil
	})

	// 创建登录 Lambda
	loginLambda := core.NewLambdaWithMiddleware(
		"login",
		LoginHandler,
		core.Logger[LoginRequest, LoginResponse]("Login"),
		validateLogin,
		core.Recovery[LoginRequest, LoginResponse](),
	)

	// 测试登录
	testCases := []LoginRequest{
		{Username: "admin", Password: "secret"},
		{Username: "admin", Password: "wrong"},
		{Username: "", Password: "secret"},
		{Username: "user", Password: "123"}, // password too short
	}

	for _, tc := range testCases {
		fmt.Printf("\n  Testing: %s / ***\n", tc.Username)
		result, err := loginLambda.Invoke(context.Background(), tc)
		if err != nil {
			fmt.Printf("    ✗ Error: %v\n", err)
		} else {
			fmt.Printf("    ✓ Success! Token: %s\n", result.Output.Token)
		}
	}
}

// demoDynamicMiddleware 动态添加中间件演示
func demoDynamicMiddleware() {
	// 创建基础 Lambda
	lambda := core.NewLambdaWithMiddleware(
		"dynamic_processor",
		SimpleBusinessHandler,
		core.Logger[string, string]("DynamicProcessor"),
	)

	fmt.Println("  Initial call:")
	result1, _ := lambda.Invoke(context.Background(), "Test 1")
	fmt.Printf("    Result: %s\n\n", result1.Output)

	// 动态添加更多中间件
	lambdaWithMore := lambda.Use(
		core.Timeout[string, string](50*time.Millisecond),
		core.Retry[string, string](2),
	)

	fmt.Println("  After adding Timeout and Retry:")
	result2, err := lambdaWithMore.Invoke(context.Background(), "Test 2")
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Printf("    Result: %s\n", result2.Output)
	}
}

// demoRetry 重试机制演示
func demoRetry() {
	retryLambda := core.NewLambdaWithMiddleware(
		"retry_demo",
		FailingHandler,
		core.Logger[int, string]("RetryDemo"),
		core.Retry[int, string](3), // 最多重试3次
	)

	fmt.Println("  Testing with input that will succeed after retries:")
	result, err := retryLambda.Invoke(context.Background(), 1)
	if err != nil {
		fmt.Printf("    Failed: %v\n", err)
	} else {
		fmt.Printf("    Success: %s\n", result.Output)
	}
}

// demoRateLimit 限流演示
func demoRateLimit() {
	// 创建限流器：每秒最多 2 个请求
	limiter := core.NewRateLimiter(2, time.Second)

	lambda := core.NewLambdaWithMiddleware(
		"rate_limited",
		SimpleBusinessHandler,
		core.RateLimit[string, string](limiter),
	)

	fmt.Println("  Sending 5 requests rapidly:")
	for i := 1; i <= 5; i++ {
		_, err := lambda.Invoke(context.Background(), fmt.Sprintf("Request %d", i))
		if err != nil {
			fmt.Printf("    Request %d: BLOCKED - %v\n", i, err)
		} else {
			fmt.Printf("    Request %d: ALLOWED\n", i)
		}
	}

	fmt.Println("\n  Waiting for rate limit window to pass...")
	time.Sleep(1100 * time.Millisecond)

	fmt.Println("  Sending another request:")
	_, err := lambda.Invoke(context.Background(), "Request after wait")
	if err != nil {
		fmt.Printf("    BLOCKED - %v\n", err)
	} else {
		fmt.Printf("    ALLOWED\n")
	}
}

// demoCircuitBreaker 熔断器演示
func demoCircuitBreaker() {
	// 简化的熔断器状态
	type circuitState struct {
		failures    int
		lastFailure time.Time
		openUntil   time.Time
	}
	cb := &circuitState{}
	maxFailures := 3
	resetTimeout := 2 * time.Second

	// 简单的熔断器中间件
	circuitBreakerMiddleware := func(ctx context.Context, input string, next core.InvokeFunc[string, string]) (string, error) {
		// 检查熔断器是否打开
		if time.Now().Before(cb.openUntil) {
			return "", fmt.Errorf("circuit breaker is OPEN")
		}

		output, err := next(ctx, input)

		if err != nil {
			cb.failures++
			cb.lastFailure = time.Now()

			// 达到失败阈值，打开熔断器
			if cb.failures >= maxFailures {
				cb.openUntil = time.Now().Add(resetTimeout)
				fmt.Printf("    [CircuitBreaker] Threshold reached, opening until %v\n", cb.openUntil.Format(time.RFC3339))
			}
			return output, err
		}

		// 成功时重置计数
		cb.failures = 0
		return output, nil
	}

	lambda := core.NewLambdaWithMiddleware(
		"circuit_breaker_demo",
		func(ctx context.Context, input string) (string, error) {
			// 模拟会失败的服务
			if input == "fail" {
				return "", errors.New("service unavailable")
			}
			return fmt.Sprintf("Processed: %s", input), nil
		},
		circuitBreakerMiddleware,
	)

	fmt.Println("  Sending requests that will fail:")
	for i := 1; i <= 4; i++ {
		_, err := lambda.Invoke(context.Background(), "fail")
		if err != nil {
			fmt.Printf("    Attempt %d: Failed\n", i)
		}
	}

	fmt.Println("\n  Circuit breaker should be OPEN now. Trying again:")
	_, err := lambda.Invoke(context.Background(), "fail")
	if err != nil {
		fmt.Printf("    Request blocked: %v\n", err)
	}

	fmt.Println("\n  Waiting for circuit breaker to reset...")
	time.Sleep(2100 * time.Millisecond)

	fmt.Println("  Trying with valid input after reset:")
	result, err := lambda.Invoke(context.Background(), "success")
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Printf("    Success: %s\n", result.Output)
	}
}

// demoCustomMiddleware 自定义中间件演示
func demoCustomMiddleware() {
	// 使用自定义的认证和审计中间件
	lambda := core.NewLambdaWithMiddleware(
		"user_service",
		GetUserHandler,
		core.Logger[int, User]("GetUser"),
		Auth[int, User]("user"),
		AuditLog[int, User](),
	)

	// 没有认证
	fmt.Println("  Request without authentication:")
	_, err := lambda.Invoke(context.Background(), 123)
	fmt.Printf("    Result: %v\n\n", err)

	// 带认证
	fmt.Println("  Request with authentication:")
	authCtx := context.WithValue(context.Background(), "user_id", 42)
	result, err := lambda.Invoke(authCtx, 123)
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Printf("    User: %s (%s)\n", result.Output.Name, result.Output.Email)
	}

	// 使用输入清理中间件
	sanitizeLambda := core.NewLambdaWithMiddleware(
		"sanitizer",
		SimpleBusinessHandler,
		SanitizeInput(),
	)

	fmt.Println("\n  Testing input sanitization:")
	longInput := string(make([]byte, 150))
	for i := range longInput {
		longInput = longInput[:i] + "A"
	}
	result2, _ := sanitizeLambda.Invoke(context.Background(), longInput)
	fmt.Printf("    Processed (truncated): %s...\n", result2.Output[:50])
}

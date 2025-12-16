package test

import (
	"context"
	"fmt"
	"github.com/ZHLX2005/minilambda/core"
	"github.com/ZHLX2005/minilambda/invoker"
	"github.com/ZHLX2005/minilambda/registry"
	"strings"
	"testing"
	"time"
)

// Person 和 PersonGreeting 类型定义
type Person struct {
	Name string
	Age  int
}

type PersonGreeting struct {
	Message string
	IsValid bool
}

func TestMain(m *testing.M) {
	// 初始化minilambda系统
	registry.RegisterAutoHandler(func() {
		// 注册测试用lambda
		registry.RegisterLambda("test_add", func(ctx context.Context, input int) (int, error) {
			return input + 1, nil
		})
		registry.RegisterLambda("test_multiply", func(ctx context.Context, input int) (int, error) {
			return input * 3, nil
		})
		registry.RegisterLambda("test_error", func(ctx context.Context, input string) (int, error) {
			return 0, fmt.Errorf("test error")
		})

		// 注册示例lambda
		registry.RegisterLambda("string_upper", func(ctx context.Context, input string) (string, error) {
			return strings.ToUpper(input), nil
		})
		registry.RegisterLambda("string_lower", func(ctx context.Context, input string) (string, error) {
			return strings.ToLower(input), nil
		})
		registry.RegisterLambda("string_reverse", func(ctx context.Context, input string) (string, error) {
			runes := []rune(input)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			return string(runes), nil
		})
		registry.RegisterLambda("math_double", func(ctx context.Context, input int) (int, error) {
			return input * 2, nil
		})
		registry.RegisterLambda("math_square", func(ctx context.Context, input int) (int, error) {
			return input * input, nil
		})
		registry.RegisterLambda("math_factorial", func(ctx context.Context, input int) (int, error) {
			if input < 0 {
				return 0, fmt.Errorf("factorial is not defined for negative numbers")
			}
			if input == 0 || input == 1 {
				return 1, nil
			}
			result := 1
			for i := 2; i <= input; i++ {
				result *= i
			}
			return result, nil
		})

		// 注册Person相关
		registry.RegisterLambda("validate_person", func(ctx context.Context, input Person) (PersonGreeting, error) {
			if input.Name == "" {
				return PersonGreeting{IsValid: false}, nil
			}
			if input.Age < 0 || input.Age > 150 {
				return PersonGreeting{IsValid: false}, nil
			}
			return PersonGreeting{
				Message: fmt.Sprintf("Valid person: %s, age %d", input.Name, input.Age),
				IsValid: true,
			}, nil
		})
		registry.RegisterLambda("create_greeting", func(ctx context.Context, input Person) (PersonGreeting, error) {
			if input.Name == "" {
				return PersonGreeting{IsValid: false}, fmt.Errorf("name cannot be empty")
			}
			var message string
			if input.Age < 18 {
				message = fmt.Sprintf("Hi %s! You're young and full of potential!", input.Name)
			} else if input.Age < 65 {
				message = fmt.Sprintf("Hello %s! You're in the prime of your life!", input.Name)
			} else {
				message = fmt.Sprintf("Greetings %s! You have a wealth of experience!", input.Name)
			}
			return PersonGreeting{
				Message: message,
				IsValid: true,
			}, nil
		})
	})
	registry.ExecuteAutoHandlers()
	m.Run()
}

func TestBasicLambdaRegistration(t *testing.T) {
	// 测试lambda注册 - 使用不同的名称避免冲突
	err := registry.RegisterLambda("test_basic_add", func(ctx context.Context, input int) (int, error) {
		return input + 1, nil
	})
	if err != nil && err.Error() != "lambda 'test_basic_add' already registered" {
		t.Fatalf("Failed to register lambda: %v", err)
	}

	// 测试获取lambda
	lambda, exists := registry.GetLambda[int, int]("test_basic_add")
	if !exists {
		t.Fatal("Lambda not found after registration")
	}

	if lambda.GetName() != "test_basic_add" {
		t.Errorf("Expected name 'test_basic_add', got '%s'", lambda.GetName())
	}
}

func TestLambdaInvocation(t *testing.T) {
	// 注册测试lambda
	registry.RegisterLambda("test_multiply", func(ctx context.Context, input int) (int, error) {
		return input * 3, nil
	})

	// 创建调用器
	inv := invoker.NewInvoker[int, int]()

	// 调用lambda
	result, err := inv.Invoke(context.Background(), "test_multiply", 5)
	if err != nil {
		t.Fatalf("Lambda invocation failed: %v", err)
	}

	if result.Output != 15 {
		t.Errorf("Expected output 15, got %d", result.Output)
	}

	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}
}

func TestStringProcessingLambdas(t *testing.T) {
	inv := invoker.NewInvoker[string, string]()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"string_upper", "hello", "HELLO"},
		{"string_lower", "WORLD", "world"},
		{"string_reverse", "abcde", "edcba"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := inv.Invoke(context.Background(), test.name, test.input)
			if err != nil {
				t.Fatalf("Lambda invocation failed: %v", err)
			}

			if result.Output != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result.Output)
			}
		})
	}
}

func TestMathLambdas(t *testing.T) {
	inv := invoker.NewInvoker[int, int]()

	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"math_double", 7, 14},
		{"math_square", 6, 36},
		{"math_factorial", 5, 120},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := inv.Invoke(context.Background(), test.name, test.input)
			if err != nil {
				t.Fatalf("Lambda invocation failed: %v", err)
			}

			if result.Output != test.expected {
				t.Errorf("Expected %d, got %d", test.expected, result.Output)
			}
		})
	}
}

func TestAsyncInvocation(t *testing.T) {
	inv := invoker.NewInvoker[string, string]()

	// 异步调用
	resultChan := inv.InvokeAsync(context.Background(), "string_upper", "async_test")

	// 等待结果
	select {
	case result := <-resultChan:
		if result.Error != nil {
			t.Fatalf("Async lambda invocation failed: %v", result.Error)
		}
		if result.Output != "ASYNC_TEST" {
			t.Errorf("Expected 'ASYNC_TEST', got '%s'", result.Output)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Async invocation timed out")
	}
}

func TestMultipleInvocations(t *testing.T) {
	inv := invoker.NewInvoker[int, int]()

	requests := map[string]int{
		"math_double": 10,
		"math_square": 3,
	}

	results := inv.InvokeMultiple(context.Background(), requests)

	// 验证结果
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	doubleResult, exists := results["math_double"]
	if !exists {
		t.Fatal("math_double result not found")
	}
	if doubleResult.Output != 20 {
		t.Errorf("Expected math_double result 20, got %d", doubleResult.Output)
	}

	squareResult, exists := results["math_square"]
	if !exists {
		t.Fatal("math_square result not found")
	}
	if squareResult.Output != 9 {
		t.Errorf("Expected math_square result 9, got %d", squareResult.Output)
	}
}

func TestPipelineInvocation(t *testing.T) {
	inv := invoker.NewInvoker[int, int]()

	inputs := []int{1, 2, 3, 4, 5}
	results, err := inv.Pipeline(context.Background(), "math_double", inputs)
	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}

	expected := []int{2, 4, 6, 8, 10}
	for i, result := range results {
		if result.Output != expected[i] {
			t.Errorf("Pipeline step %d: expected %d, got %d", i, expected[i], result.Output)
		}
	}
}

func TestLambdaOptions(t *testing.T) {
	// 注册带选项的lambda
	lambda := core.NewLambda("test_with_options", func(ctx context.Context, input string) (string, error) {
		return input + "_processed", nil
	},
		core.WithTimeout(1*time.Second),
		core.WithEnableMetrics(true),
		core.WithComponentType("TestProcessor"),
	)

	// 验证选项设置
	options := lambda.GetOptions()
	if options.Timeout != 1*time.Second {
		t.Errorf("Expected timeout 1s, got %v", options.Timeout)
	}
	if !options.EnableMetrics {
		t.Error("Expected EnableMetrics to be true")
	}
	if options.ComponentType != "TestProcessor" {
		t.Errorf("Expected component type 'TestProcessor', got '%s'", options.ComponentType)
	}

	// 测试调用
	result, err := lambda.Invoke(context.Background(), "test")
	if err != nil {
		t.Fatalf("Lambda invocation failed: %v", err)
	}
	if result.Output != "test_processed" {
		t.Errorf("Expected 'test_processed', got '%s'", result.Output)
	}
}

func TestComplexDataStructures(t *testing.T) {
	inv := invoker.NewInvoker[Person, PersonGreeting]()

	person := Person{
		Name: "Alice",
		Age:  25,
	}

	result, err := inv.Invoke(context.Background(), "validate_person", person)
	if err != nil {
		t.Fatalf("Lambda invocation failed: %v", err)
	}

	if !result.Output.IsValid {
		t.Error("Expected person to be valid")
	}

	greetingResult, err := inv.Invoke(context.Background(), "create_greeting", person)
	if err != nil {
		t.Fatalf("Greeting lambda invocation failed: %v", err)
	}

	if !greetingResult.Output.IsValid {
		t.Error("Expected greeting to be valid")
	}
	if greetingResult.Output.Message == "" {
		t.Error("Expected greeting message to be non-empty")
	}
}

func TestErrorHandling(t *testing.T) {
	inv := invoker.NewInvoker[string, int]()

	// 尝试调用不存在的lambda
	_, err := inv.Invoke(context.Background(), "non_existent", "test")
	if err == nil {
		t.Error("Expected error for non-existent lambda")
	}

	// 测试lambda内部错误 - 使用不同的名称避免重复注册
	err = registry.RegisterLambda("test_error_v2", func(ctx context.Context, input string) (int, error) {
		return 0, fmt.Errorf("test error")
	})
	if err != nil && err.Error() != "lambda 'test_error_v2' already registered" {
		t.Fatalf("Failed to register lambda: %v", err)
	}

	result, err := inv.Invoke(context.Background(), "test_error_v2", "test")
	if err != nil {
		// 如果有错误，检查是否是lambda内部错误
		if err.Error() == "test error" {
			// 这是预期的lambda内部错误，创建结果结构
			var zero int
			result = &core.LambdaResult[int]{
				Output:    zero,
				Error:     err,
				Duration:  0,
				Timestamp: time.Now(),
			}
		} else {
			t.Fatalf("Lambda invocation failed: %v", err)
		}
	}

	if result.Error == nil {
		t.Error("Expected lambda to return error")
	}
	if result.Error.Error() != "test error" {
		t.Errorf("Expected error 'test error', got '%v'", result.Error)
	}
}

func TestLambdaMetrics(t *testing.T) {
	lambda := core.NewLambda("test_metrics", func(ctx context.Context, input int) (int, error) {
		return input * 2, nil
	}, core.WithEnableMetrics(true))

	// 调用几次
	for i := 0; i < 3; i++ {
		_, err := lambda.Invoke(context.Background(), i)
		if err != nil {
			t.Fatalf("Lambda invocation failed: %v", err)
		}
	}

	// 检查指标
	metrics := lambda.GetMetrics()
	if metrics.TotalInvocations != 3 {
		t.Errorf("Expected 3 total invocations, got %d", metrics.TotalInvocations)
	}
	if metrics.SuccessInvocations != 3 {
		t.Errorf("Expected 3 success invocations, got %d", metrics.SuccessInvocations)
	}
	if metrics.ErrorInvocations != 0 {
		t.Errorf("Expected 0 error invocations, got %d", metrics.ErrorInvocations)
	}
}

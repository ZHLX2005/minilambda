package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"minilambda/core"
	"minilambda/example"
	"minilambda/invoker"
	"minilambda/registry"
)

func init() {
	// 注册示例lambda
	example.RegisterExampleLambdas()
}

func main() {
	fmt.Println("=== MiniLambda Demo ===")

	// 初始化minilambda
	registry.ExecuteAutoHandlers()

	// 基本用法示例
	basicUsageDemo()

	// 异步调用示例
	asyncDemo()

	// 批量调用示例
	batchDemo()

	// 管道调用示例
	pipelineDemo()

	// 链式调用示例
	chainDemo()

	// 指标监控示例
	metricsDemo()

	fmt.Println("=== Demo Completed ===")
}

func basicUsageDemo() {
	fmt.Println("\n--- Basic Usage Demo ---")

	// 创建调用器
	inv := invoker.NewInvoker[string, string]()

	// 调用字符串处理lambda
	result, err := inv.Invoke(context.Background(), "string_upper", "hello world")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Input: 'hello world'\n")
	fmt.Printf("Lambda: string_upper\n")
	fmt.Printf("Output: '%s'\n", result.Output)
	fmt.Printf("Duration: %v\n", result.Duration)
}

func asyncDemo() {
	fmt.Println("\n--- Async Demo ---")

	inv := invoker.NewInvoker[string, string]()

	// 异步调用
	start := time.Now()
	resultChan := inv.InvokeAsync(context.Background(), "string_reverse", "async")

	// 可以在等待结果时做其他事情
	fmt.Println("Lambda is running asynchronously...")
	time.Sleep(50 * time.Millisecond)

	// 获取结果
	result := <-resultChan
	fmt.Printf("Async result: '%s'\n", result.Output)
	fmt.Printf("Total time: %v\n", time.Since(start))
}

func batchDemo() {
	fmt.Println("\n--- Batch Demo ---")

	inv := invoker.NewInvoker[int, int]()

	// 批量调用
	requests := map[string]int{
		"math_double":   10,
		"math_square":   5,
		"math_factorial": 4,
	}

	results := inv.InvokeMultiple(context.Background(), requests)
	fmt.Println("Batch results:")
	for name, result := range results {
		fmt.Printf("  %s: %d (error: %v)\n", name, result.Output, result.Error)
	}
}

func pipelineDemo() {
	fmt.Println("\n--- Pipeline Demo ---")

	inv := invoker.NewInvoker[int, int]()

	inputs := []int{1, 2, 3, 4, 5}
	results, err := inv.Pipeline(context.Background(), "math_double", inputs)
	if err != nil {
		log.Printf("Pipeline error: %v", err)
		return
	}

	fmt.Println("Pipeline results:")
	for i, result := range results {
		fmt.Printf("  Step %d: %d -> %d\n", i+1, inputs[i], result.Output)
	}
}

func chainDemo() {
	fmt.Println("\n--- Chain Demo ---")

	// 手动模拟链式调用：int -> string -> string -> string

	// 手动模拟后续步骤
	ctx := context.Background()
	inv1 := invoker.NewInvoker[int, string]()
	result1, err := inv1.Invoke(ctx, "int_to_string", 42)
	if err != nil {
		log.Printf("Chain step 1 error: %v", err)
		return
	}

	inv2 := invoker.NewInvoker[string, string]()
	result2, err := inv2.Invoke(ctx, "string_upper", result1.Output)
	if err != nil {
		log.Printf("Chain step 2 error: %v", err)
		return
	}

	fmt.Printf("Chain result: 42 -> %s -> %s\n", result1.Output, result2.Output)
}

func metricsDemo() {
	fmt.Println("\n--- Metrics Demo ---")

	// 创建带指标收集的lambda
	lambda := core.NewLambda("demo_metrics", func(ctx context.Context, input int) (int, error) {
		// 模拟一些工作
		time.Sleep(10 * time.Millisecond)
		return input * input, nil
	}, core.WithEnableMetrics(true))

	// 调用多次
	for i := 1; i <= 5; i++ {
		_, err := lambda.Invoke(context.Background(), i)
		if err != nil {
			log.Printf("Lambda invocation error: %v", err)
			return
		}
	}

	// 获取指标
	metrics := lambda.GetMetrics()
	fmt.Printf("Lambda metrics:\n")
	fmt.Printf("  Total invocations: %d\n", metrics.TotalInvocations)
	fmt.Printf("  Success invocations: %d\n", metrics.SuccessInvocations)
	fmt.Printf("  Error invocations: %d\n", metrics.ErrorInvocations)
	fmt.Printf("  Average duration: %v\n", metrics.AverageDuration)
	fmt.Printf("  Last invocation: %v\n", metrics.LastInvocationTime.Format(time.RFC3339))
}

// 演示自定义lambda注册
func registerCustomLambda() {
	fmt.Println("\n--- Custom Lambda Registration ---")

	// 注册自定义lambda
	err := registry.RegisterLambda("custom_greet", func(ctx context.Context, input string) (string, error) {
		return fmt.Sprintf("Greetings, %s! This is a custom lambda.", input), nil
	})
	if err != nil {
		log.Printf("Failed to register custom lambda: %v", err)
		return
	}

	// 使用自定义lambda
	inv := invoker.NewInvoker[string, string]()
	result, err := inv.Invoke(context.Background(), "custom_greet", "Developer")
	if err != nil {
		log.Printf("Custom lambda error: %v", err)
		return
	}

	fmt.Printf("Custom lambda result: %s\n", result.Output)
}
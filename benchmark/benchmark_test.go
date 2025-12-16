package benchmark

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"minilambda/core"
	"minilambda/invoker"
	"minilambda/registry"
)

// 基准测试用的简单函数
func simpleAdd(x int) int {
	return x + 1
}

func simpleMultiply(x int) int {
	return x * 2
}

func simpleStringUpper(s string) string {
	return s + "_processed"
}

// Lambda版本的函数
func lambdaAdd(ctx context.Context, x int) (int, error) {
	return x + 1, nil
}

func lambdaMultiply(ctx context.Context, x int) (int, error) {
	return x * 2, nil
}

func lambdaStringUpper(ctx context.Context, s string) (string, error) {
	return s + "_processed", nil
}

func init() {
	registry.RegisterAutoHandler(func() {
		registry.RegisterLambda("benchmark_add", lambdaAdd)
		registry.RegisterLambda("benchmark_multiply", lambdaMultiply)
		registry.RegisterLambda("benchmark_string_upper", lambdaStringUpper)
	})
	registry.ExecuteAutoHandlers()
}

// 基准测试：直接函数调用 vs Lambda调用
func BenchmarkDirectAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = simpleAdd(i)
	}
}

func BenchmarkLambdaAdd(b *testing.B) {
	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := inv.Invoke(ctx, "benchmark_add", i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDirectMultiply(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = simpleMultiply(i)
	}
}

func BenchmarkLambdaMultiply(b *testing.B) {
	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := inv.Invoke(ctx, "benchmark_multiply", i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDirectStringUpper(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = simpleStringUpper(fmt.Sprintf("test_%d", i))
	}
}

func BenchmarkLambdaStringUpper(b *testing.B) {
	inv := invoker.NewInvoker[string, string]()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := inv.Invoke(ctx, "benchmark_string_upper", fmt.Sprintf("test_%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 直接使用Lambda实例（不通过注册中心）
func BenchmarkDirectLambdaInstance(b *testing.B) {
	lambda := core.NewLambda("direct_lambda", lambdaAdd)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := lambda.Invoke(ctx, i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 并发基准测试
func BenchmarkDirectAddConcurrent(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = simpleAdd(i)
			i++
		}
	})
}

func BenchmarkLambdaAddConcurrent(b *testing.B) {
	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, err := inv.Invoke(ctx, "benchmark_add", i)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// 内存分配基准测试
func BenchmarkDirectAddAllocs(b *testing.B) {
	var result int
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result = simpleAdd(i)
	}
	_ = result
}

func BenchmarkLambdaAddAllocs(b *testing.B) {
	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result, err := inv.Invoke(ctx, "benchmark_add", i)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}
}

// 压力测试：高并发场景
func BenchmarkHighConcurrencyDirect(b *testing.B) {
	var wg sync.WaitGroup
	goroutines := runtime.NumCPU() * 4
	iterations := b.N / goroutines

	b.ResetTimer()

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_ = simpleAdd(i)
			}
		}()
	}

	wg.Wait()
}

func BenchmarkHighConcurrencyLambda(b *testing.B) {
	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()
	var wg sync.WaitGroup
	goroutines := runtime.NumCPU() * 4
	iterations := b.N / goroutines

	b.ResetTimer()

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_, err := inv.Invoke(ctx, "benchmark_add", start+i)
				if err != nil {
					b.Fatal(err)
				}
			}
		}(g * iterations)
	}

	wg.Wait()
}

// 不同操作的基准测试
func BenchmarkDifferentOperations(b *testing.B) {
	operations := []string{"add", "multiply", "string"}

	for _, op := range operations {
		b.Run(fmt.Sprintf("direct_%s", op), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				switch op {
				case "add":
					_ = simpleAdd(i)
				case "multiply":
					_ = simpleMultiply(i)
				case "string":
					_ = simpleStringUpper(fmt.Sprintf("test_%d", i))
				}
			}
		})

		b.Run(fmt.Sprintf("lambda_%s", op), func(b *testing.B) {
			var inv interface{}
			var name string

			switch op {
			case "add":
				inv = invoker.NewInvoker[int, int]()
				name = "benchmark_add"
			case "multiply":
				inv = invoker.NewInvoker[int, int]()
				name = "benchmark_multiply"
			case "string":
				inv = invoker.NewInvoker[string, string]()
				name = "benchmark_string_upper"
			}

			ctx := context.Background()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				switch op {
				case "add":
					result, err := inv.(*invoker.Invoker[int, int]).Invoke(ctx, name, i)
					if err != nil {
						b.Fatal(err)
					}
					_ = result
				case "multiply":
					result, err := inv.(*invoker.Invoker[int, int]).Invoke(ctx, name, i)
					if err != nil {
						b.Fatal(err)
					}
					_ = result
				case "string":
					result, err := inv.(*invoker.Invoker[string, string]).Invoke(ctx, name, fmt.Sprintf("test_%d", i))
					if err != nil {
						b.Fatal(err)
					}
					_ = result
				}
			}
		})
	}
}

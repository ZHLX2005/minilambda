package benchmark

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"minilambda/invoker"
	"minilambda/registry"
)

// 复杂计算函数用于压力测试
func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func lambdaFibonacci(ctx context.Context, n int) (int, error) {
	return fibonacci(n), nil
}

func init() {
	registry.RegisterLambda("stress_fibonacci", lambdaFibonacci)
}

// 压力测试：大量并发调用
func TestStressHighConcurrency(t *testing.T) {
	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()

	numGoroutines := 100
	numCalls := 1000
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errorCount := 0

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numCalls; j++ {
				_, err := inv.Invoke(ctx, "benchmark_add", id*numCalls+j)
				mu.Lock()
				if err != nil {
					errorCount++
				} else {
					successCount++
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalCalls := numGoroutines * numCalls
	t.Logf("压力测试结果:")
	t.Logf("  总调用数: %d", totalCalls)
	t.Logf("  成功调用: %d", successCount)
	t.Logf("  失败调用: %d", errorCount)
	t.Logf("  总耗时: %v", duration)
	t.Logf("  平均QPS: %.2f", float64(totalCalls)/duration.Seconds())
	t.Logf("  平均延迟: %v", duration/time.Duration(totalCalls))
}

// 内存压力测试
func TestStressMemoryUsage(t *testing.T) {
	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 执行大量调用
	numCalls := 100000
	for i := 0; i < numCalls; i++ {
		_, err := inv.Invoke(ctx, "benchmark_add", i)
		if err != nil {
			t.Fatal(err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	t.Logf("内存使用情况:")
	t.Logf("  调用次数: %d", numCalls)
	t.Logf("  分配内存增长: %d bytes", m2.TotalAlloc-m1.TotalAlloc)
	t.Logf("  平均每次调用分配: %.2f bytes", float64(m2.TotalAlloc-m1.TotalAlloc)/float64(numCalls))
	t.Logf("  堆对象增长: %d", m2.HeapObjects-m1.HeapObjects)
}

// CPU密集型压力测试
func TestStressCPUIntensive(t *testing.T) {
	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()

	numCalls := 1000
	var wg sync.WaitGroup
	var totalDuration time.Duration
	var mu sync.Mutex

	start := time.Now()

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			callStart := time.Now()
			_, err := inv.Invoke(ctx, "stress_fibonacci", n%20) // 限制fibonacci输入避免太慢
			if err != nil {
				t.Fatal(err)
			}
			callDuration := time.Since(callStart)

			mu.Lock()
			totalDuration += callDuration
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	overallDuration := time.Since(start)

	t.Logf("CPU密集型压力测试:")
	t.Logf("  总调用数: %d", numCalls)
	t.Logf("  总耗时: %v", overallDuration)
	t.Logf("  平均每次调用耗时: %v", totalDuration/time.Duration(numCalls))
	t.Logf("  并发效率: %.2f%%", float64(totalDuration)/float64(overallDuration)*100)
}

// 长时间运行稳定性测试
func TestStressLongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过长时间运行测试")
	}

	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()

	duration := 10 * time.Second
	endTime := time.Now().Add(duration)

	var callCount int64
	var errorCount int64
	var mu sync.Mutex

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	stats := make([]struct {
		calls  int64
		errors int64
		qps    float64
	}, 0)

	for time.Now().Before(endTime) {
		select {
		case <-ticker.C:
			// 每秒统计一次
			mu.Lock()
			currentCalls := callCount
			currentErrors := errorCount
			mu.Unlock()

			qps := float64(currentCalls) / time.Since(time.Now().Add(-duration)).Seconds()
			stats = append(stats, struct {
				calls  int64
				errors int64
				qps    float64
			}{currentCalls, currentErrors, qps})

			t.Logf("运行 %v: 调用 %d, 错误 %d, QPS %.2f",
				time.Since(time.Now().Add(-duration)), currentCalls, currentErrors, qps)
		default:
			// 持续调用
			go func() {
				_, err := inv.Invoke(ctx, "benchmark_add", 1)
				mu.Lock()
				callCount++
				if err != nil {
					errorCount++
				}
				mu.Unlock()
			}()
		}
	}

	// 等待所有调用完成
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	finalCalls := callCount
	finalErrors := errorCount
	mu.Unlock()

	t.Logf("长时间运行测试结果:")
	t.Logf("  总调用数: %d", finalCalls)
	t.Logf("  总错误数: %d", finalErrors)
	t.Logf("  错误率: %.4f%%", float64(finalErrors)/float64(finalCalls)*100)
	t.Logf("  平均QPS: %.2f", float64(finalCalls)/duration.Seconds())
}

// 资源泄漏检测
func TestStressResourceLeak(t *testing.T) {
	inv := invoker.NewInvoker[int, int]()
	ctx := context.Background()

	var m1, m2, m3 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 第一轮调用
	for i := 0; i < 10000; i++ {
		_, err := inv.Invoke(ctx, "benchmark_add", i)
		if err != nil {
			t.Fatal(err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// 第二轮调用
	for i := 0; i < 10000; i++ {
		_, err := inv.Invoke(ctx, "benchmark_add", i)
		if err != nil {
			t.Fatal(err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m3)

	t.Logf("资源泄漏检测:")
	t.Logf("  第一轮内存增长: %d bytes", m2.TotalAlloc-m1.TotalAlloc)
	t.Logf("  第二轮内存增长: %d bytes", m3.TotalAlloc-m2.TotalAlloc)
	t.Logf("  Goroutine数量: %d", runtime.NumGoroutine())

	// 检查是否有明显的内存泄漏
	if m3.TotalAlloc-m2.TotalAlloc > (m2.TotalAlloc-m1.TotalAlloc)*2 {
		t.Errorf("可能存在内存泄漏，第二轮内存增长明显大于第一轮")
	}
}

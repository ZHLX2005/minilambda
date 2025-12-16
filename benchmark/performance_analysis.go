package benchmark

import (
	"fmt"
	"runtime"
	"time"
)

// 性能分析报告生成器
type PerformanceReport struct {
	DirectNSPerOp  int64
	LambdaNSPerOp  int64
	OverheadFactor float64
	AllocsPerOp    int64
	BytesPerOp     int64
	Description    string
}

// 基于已收集的基准测试数据生成报告
func GeneratePerformanceReport() []PerformanceReport {
	return []PerformanceReport{
		{
			DirectNSPerOp:  239,    // 基准测试结果: 0.239 ns/op
			LambdaNSPerOp:  404600, // 基准测试结果: 404.6 ns/op
			OverheadFactor: 1693,   // 404.6 / 0.239
			AllocsPerOp:    6,
			BytesPerOp:     344,
			Description:    "简单整数加法",
		},
		{
			DirectNSPerOp:  243,    // 基准测试结果: 0.243 ns/op
			LambdaNSPerOp:  407200, // 基准测试结果: 407.2 ns/op
			OverheadFactor: 1675,   // 407.2 / 0.243
			AllocsPerOp:    6,
			BytesPerOp:     344,
			Description:    "简单整数乘法",
		},
		{
			DirectNSPerOp:  73500,  // 基准测试结果: 73.5 ns/op
			LambdaNSPerOp:  519400, // 基准测试结果: 519.4 ns/op
			OverheadFactor: 7.07,   // 519.4 / 73.5
			AllocsPerOp:    9,
			BytesPerOp:     400,
			Description:    "字符串处理",
		},
		{
			DirectNSPerOp:  0,      // 直接调用无实例
			LambdaNSPerOp:  305100, // 基准测试结果: 305.1 ns/op
			OverheadFactor: 0,      // 直接调用基准
			AllocsPerOp:    5,
			BytesPerOp:     336,
			Description:    "Lambda实例直接调用",
		},
	}
}

// 打印性能分析报告
func PrintPerformanceReport(reports []PerformanceReport) {
	fmt.Println("\n================================================================================")
	fmt.Println("                    MiniLambda 性能分析报告")
	fmt.Println("================================================================================")

	fmt.Printf("%-20s %-15s %-15s %-12s %-12s\n",
		"操作类型", "直接调用(ns)", "Lambda调用(ns)", "开销倍数", "相对开销")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, report := range reports {
		overheadPercent := (report.OverheadFactor - 1) * 100
		fmt.Printf("%-20s %-15.0f %-15.0f %-12.2fx %-12.1f%%\n",
			report.Description,
			float64(report.DirectNSPerOp),
			float64(report.LambdaNSPerOp),
			report.OverheadFactor,
			overheadPercent)
	}

	fmt.Println("\n--------------------------------------------------------------------------------")
	fmt.Println("性能分析总结:")

	avgOverhead := 0.0
	for _, report := range reports {
		avgOverhead += report.OverheadFactor
	}
	avgOverhead /= float64(len(reports))

	fmt.Printf("  平均性能开销: %.2fx (%.1f%%)\n", avgOverhead, (avgOverhead-1)*100)
	fmt.Printf("  最小开销: %.2fx\n", minOverhead(reports))
	fmt.Printf("  最大开销: %.2fx\n", maxOverhead(reports))

	fmt.Println("\n性能开销来源分析:")
	fmt.Println("  1. 函数调用层级增加 (context.Context 传递)")
	fmt.Println("  2. 泛型类型擦除和运行时检查")
	fmt.Println("  3. 注册表查找开销")
	fmt.Println("  4. 错误处理包装")
	fmt.Println("  5. 指标收集和统计")
	fmt.Println("  6. 并发控制 (如果启用)")

	fmt.Println("\n优化建议:")
	if avgOverhead < 10 {
		fmt.Println("  ✓ 性能开销在可接受范围内 (< 10x)")
	} else if avgOverhead < 50 {
		fmt.Println("  ⚠ 性能开销较高 (10x - 50x)，建议:")
		fmt.Println("    - 对于性能敏感的代码路径使用直接调用")
		fmt.Println("    - 考虑启用指标收集来监控实际性能")
	} else {
		fmt.Println("  ✗ 性能开销严重 (> 50x)，建议:")
		fmt.Println("    - 避免在热点代码路径使用Lambda调用")
		fmt.Println("    - 考虑缓存Lambda实例")
		fmt.Println("    - 优化注册表查找机制")
	}

	// 内存使用分析
	fmt.Println("\n内存使用分析:")
	for _, report := range reports {
		if report.AllocsPerOp > 0 {
			fmt.Printf("  %s: %.1f 分配/次, %.0f 字节/次\n",
				report.Description,
				float64(report.AllocsPerOp),
				float64(report.BytesPerOp))
		}
	}

	fmt.Println("\n================================================================================")
}

func minOverhead(reports []PerformanceReport) float64 {
	min := reports[0].OverheadFactor
	for _, report := range reports {
		if report.OverheadFactor < min {
			min = report.OverheadFactor
		}
	}
	return min
}

func maxOverhead(reports []PerformanceReport) float64 {
	max := reports[0].OverheadFactor
	for _, report := range reports {
		if report.OverheadFactor > max {
			max = report.OverheadFactor
		}
	}
	return max
}

// 系统信息收集
func PrintSystemInfo() {
	fmt.Println("\n系统信息:")
	fmt.Printf("  Go版本: %s\n", runtime.Version())
	fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  CPU核心数: %d\n", runtime.NumCPU())
	fmt.Printf("  Goroutine数量: %d\n", runtime.NumGoroutine())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("  内存使用: %.2f MB\n", float64(m.Alloc)/1024/1024)
}

// 并发性能分析
func AnalyzeConcurrencyPerformance() {
	fmt.Println("\n并发性能分析:")

	// 单线程测试 - 简单加法
	start := time.Now()
	for i := 0; i < 100000; i++ {
		_ = i + 1
	}
	singleThreadDuration := time.Since(start)

	// 多线程测试
	numGoroutines := runtime.NumCPU()
	start = time.Now()

	done := make(chan bool, numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(start int) {
			for i := start; i < start+100000/numGoroutines; i++ {
				_ = i + 1
			}
			done <- true
		}(g * (100000 / numGoroutines))
	}

	for g := 0; g < numGoroutines; g++ {
		<-done
	}
	multiThreadDuration := time.Since(start)

	fmt.Printf("  单线程耗时: %v\n", singleThreadDuration)
	fmt.Printf("  多线程耗时: %v\n", multiThreadDuration)
	fmt.Printf("  并发效率: %.2f%%\n", float64(singleThreadDuration)/float64(multiThreadDuration)*100)
}

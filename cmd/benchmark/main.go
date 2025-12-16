package main

import (
	"fmt"

	"minilambda/benchmark"
)

func main() {
	fmt.Println("开始MiniLambda性能基准测试...")

	// 打印系统信息
	benchmark.PrintSystemInfo()

	// 生成并打印性能报告
	reports := benchmark.GeneratePerformanceReport()
	benchmark.PrintPerformanceReport(reports)

	// 并发性能分析
	benchmark.AnalyzeConcurrencyPerformance()

	fmt.Println("\n基准测试完成！")
	fmt.Println("\n如需运行完整压力测试，请执行:")
	fmt.Println("  go test -v ./benchmark/... -run=Stress")
	fmt.Println("  go test -v ./benchmark/... -bench=. -benchmem")
}
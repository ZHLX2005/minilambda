package minilambda

import (
	"github.com/ZHLX2005/minilambda/registry"
)

// Init 初始化minilambda系统
func Init() {
	// 执行所有自动注册处理函数
	registry.ExecuteAutoHandlers()
}

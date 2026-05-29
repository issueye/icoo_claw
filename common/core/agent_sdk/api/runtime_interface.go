package api

import (
	"context"

	"icoo_claw/common/core/agent_sdk/config"
	"icoo_claw/common/core/agent_sdk/message"
	"icoo_claw/common/core/agent_sdk/sandbox"
)

// RuntimeInterface 抽象了 Runtime 的全部公共操作，
// 允许上层调用方和测试代码通过接口解耦对具体实现的直接依赖。
//
// 典型用法：
//
//	func NewService(rt api.RuntimeInterface) *Service { ... }
type RuntimeInterface interface {
	// Run 同步执行推理管道，返回最终响应。
	Run(ctx context.Context, req Request) (*Response, error)

	// RunStream 异步执行推理管道，通过 channel 以 Anthropic SSE 协议推送事件。
	RunStream(ctx context.Context, req Request) (<-chan StreamEvent, error)

	// RunStreamWithProtocol 与 RunStream 相同，但允许指定自定义流协议编码器。
	// protocol 为 nil 时退回到 Anthropic SSE 协议（与 RunStream 兼容）。
	RunStreamWithProtocol(ctx context.Context, req Request, protocol StreamProtocol) (<-chan StreamEvent, error)

	// Config 返回当前项目配置快照（只读）。
	Config() *config.Settings

	// Settings 返回合并后的完整 settings.json 快照（只读）。
	Settings() *config.Settings

	// Sandbox 暴露底层沙箱管理器。
	Sandbox() *sandbox.Manager

	// SessionHistory 返回指定会话的消息历史快照。
	SessionHistory(sessionID string) ([]message.Message, bool)

	// Close 释放运行时持有的资源，等待正在执行的 Run 调用完成后再返回。
	Close() error
}

// 编译期断言：*Runtime 必须满足 RuntimeInterface。
// 若 Runtime 新增/删除公共方法且未同步更新接口，编译将立即失败。
var _ RuntimeInterface = (*Runtime)(nil)

package model

import "time"

// SessionRecord 是会话的 GORM 持久化模型。
//
// 设计要点：
// - session_id 使用业务生成的稳定 ID 作为主键，便于 API、消息、运行记录统一引用。
// - revision 是消息快照的乐观锁版本，Append/Replace 消息时递增，用于避免 agent 并发写入互相覆盖。
// - metadata 以 JSON 文本保存，保留 agent 编排层的扩展字段；核心查询字段 user_id/agent_id/status 单独列化并建立索引。
// - updated_at 同时承担会话列表排序游标语义，因此和 user_id、agent_id、status 组成复合索引。
type SessionRecord struct {
	SessionID    string    `gorm:"primaryKey;column:session_id;size:96"`
	UserID       string    `gorm:"column:user_id;size:128;index:idx_sessions_user_updated,priority:1"`
	AgentID      string    `gorm:"column:agent_id;size:128;index:idx_sessions_agent_updated,priority:1"`
	Title        string    `gorm:"column:title;size:256"`
	Status       string    `gorm:"column:status;size:32;not null;index:idx_sessions_status_updated,priority:1"`
	MetadataJSON string    `gorm:"column:metadata;type:text;not null"`
	Revision     int64     `gorm:"column:revision;not null;default:0"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null;index:idx_sessions_user_updated,priority:2;index:idx_sessions_agent_updated,priority:2;index:idx_sessions_status_updated,priority:2;index:idx_sessions_updated"`
}

func (SessionRecord) TableName() string { return "sessions" }

// MessageRecord 是会话消息的 GORM 持久化模型。
//
// 设计要点：
// - id 使用消息业务 ID 作为主键，方便 before/after 游标定位；position 保存会话内追加顺序。
// - session_id + position 唯一，保证同一会话消息顺序稳定且可高效 tail/before/after 分页。
// - content_blocks/tool_calls/metadata 使用 JSON 文本保存，适配多模态内容、工具调用和未来扩展。
type MessageRecord struct {
	ID                string    `gorm:"primaryKey;column:id;size:96"`
	SessionID         string    `gorm:"column:session_id;size:96;not null;uniqueIndex:idx_messages_session_position,priority:1;index:idx_messages_session_created,priority:1"`
	Position          int64     `gorm:"column:position;not null;uniqueIndex:idx_messages_session_position,priority:2"`
	Role              string    `gorm:"column:role;size:32;not null"`
	Content           string    `gorm:"column:content;type:text;not null"`
	ContentBlocksJSON string    `gorm:"column:content_blocks;type:text;not null"`
	ToolCallsJSON     string    `gorm:"column:tool_calls;type:text;not null"`
	MetadataJSON      string    `gorm:"column:metadata;type:text;not null"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;index:idx_messages_session_created,priority:2"`
}

func (MessageRecord) TableName() string { return "session_messages" }

// RunRecord 是一次 agent run 的 GORM 持久化模型。
//
// 设计要点：
// - run_id 作为主键，对应一次模型调用、工具链执行或 agent 推理过程。
// - request_id 建索引，用于从网关请求追踪到具体 run。
// - usage/metadata 使用 JSON 文本保存，兼容不同模型供应商的 token、费用、延迟等统计结构。
type RunRecord struct {
	ID           string     `gorm:"primaryKey;column:id;size:96"`
	SessionID    string     `gorm:"column:session_id;size:96;not null;uniqueIndex:idx_runs_session_position,priority:1;index:idx_runs_session_started,priority:1"`
	Position     int64      `gorm:"column:position;not null;uniqueIndex:idx_runs_session_position,priority:2"`
	RequestID    string     `gorm:"column:request_id;size:128;index:idx_runs_request"`
	Status       string     `gorm:"column:status;size:32;not null;index:idx_runs_status"`
	StopReason   string     `gorm:"column:stop_reason;size:64"`
	Error        string     `gorm:"column:error;type:text"`
	UsageJSON    string     `gorm:"column:usage;type:text;not null"`
	MetadataJSON string     `gorm:"column:metadata;type:text;not null"`
	StartedAt    time.Time  `gorm:"column:started_at;not null;index:idx_runs_session_started,priority:2"`
	CompletedAt  *time.Time `gorm:"column:completed_at"`
}

func (RunRecord) TableName() string { return "session_runs" }

// RunEventRecord 是 agent run 流式事件的 GORM 持久化模型。
//
// 设计要点：
// - id 使用业务事件 ID 作为主键，支持幂等写入和问题排查。
// - session_id + run_id + sequence 是主要读取路径，按事件序号稳定回放流式输出。
// - payload/metadata 使用 JSON 文本保存，兼容 delta、tool_call、usage、error 等不同事件载荷。
type RunEventRecord struct {
	ID           string    `gorm:"primaryKey;column:id;size:96"`
	SessionID    string    `gorm:"column:session_id;size:96;not null;index:idx_run_events_session_run_sequence,priority:1"`
	RunID        string    `gorm:"column:run_id;size:96;not null;index:idx_run_events_session_run_sequence,priority:2"`
	Type         string    `gorm:"column:type;size:64;not null"`
	Sequence     int64     `gorm:"column:sequence;not null;index:idx_run_events_session_run_sequence,priority:3"`
	PayloadJSON  string    `gorm:"column:payload;type:text;not null"`
	MetadataJSON string    `gorm:"column:metadata;type:text;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
}

func (RunEventRecord) TableName() string { return "session_run_events" }

package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/chnxq/x-crud/entgo/mixin"
)

// Task holds the schema definition for the Task entity.
type Task struct {
	ent.Schema
}

func (Task) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_tasks",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("任务表"),
	}
}

// Fields of the Task.
func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.String("task_name").
			NotEmpty().
			MaxLen(50).
			Comment("任务名称"),

		field.Uint64("group_id").
			Positive().
			Comment("任务分组ID"),

		field.Enum("task_type").
			NamedValues(
				"Function", "FUNCTION",
				"API", "API",
				"External", "EXTERNAL",
			).
			Default("FUNCTION").
			Comment("任务类型：1、函数。2、接口。3、外部执行"),

		field.String("cron_expression").
			Optional().
			Nillable().
			MaxLen(30).
			Comment("cron表达式"),

		field.String("invoke_target").
			Optional().
			Nillable().
			MaxLen(255).
			Comment("调用目标"),

		field.String("args").
			Optional().
			Nillable().
			MaxLen(255).
			Comment("目标参数"),

		field.Uint32("retry").
			Default(0).
			Comment("重试次数(最大5,0表示不重试)"),

		field.Bool("concurrent").
			Default(false).
			Comment("是否并发：1、是。0、否"),

		field.Uint32("entry_id").
			Optional().
			Nillable().
			Comment("启动时返回的ID"),

		field.Enum("status").
			NamedValues(
				"Stopped", "STOPPED",
				"Running", "RUNNING",
				"Disabled", "DISABLED",
			).
			Default("STOPPED").
			Comment("任务状态：停止、运行中、禁用"),
	}
}

// Mixin of the Task.
func (Task) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint64]{},
	}
}

// Indexes of the Task.
func (Task) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "task_name").
			StorageKey("idx_sys_task_tenant_name"),
		index.Fields("tenant_id", "group_id", "task_name").
			Unique().
			StorageKey("uix_sys_task_tenant_group_name"),
		index.Fields("tenant_id", "group_id").
			StorageKey("idx_sys_task_tenant_group"),
		index.Fields("tenant_id", "task_type").
			StorageKey("idx_sys_task_tenant_type"),
		index.Fields("tenant_id", "cron_expression").
			StorageKey("idx_sys_task_tenant_cron_expression"),
		index.Fields("tenant_id", "status").
			StorageKey("idx_sys_task_tenant_status"),
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_sys_task_tenant_created_at"),
	}
}

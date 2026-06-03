package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/chnxq/x-crud/entgo/mixin"
)

// TaskLog holds the schema definition for the TaskLog entity.
type TaskLog struct {
	ent.Schema
}

func (TaskLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_task_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("任务执行日志表"),
	}
}

func (TaskLog) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("task_id").
			Optional().
			Nillable().
			Comment("任务ID"),

		field.String("input").
			Optional().
			Nillable().
			MaxLen(255).
			Comment("执行参数"),

		field.String("output").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.MySQL:    "text",
				dialect.Postgres: "text",
			}).
			Comment("输出结果"),

		field.String("error").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.MySQL:    "text",
				dialect.Postgres: "text",
			}).
			Comment("错误信息"),

		field.Enum("status").
			NamedValues(
				"Failure", "FAILURE",
				"Success", "SUCCESS",
			).
			Default("SUCCESS").
			Comment("状态：1、成功。0、失败"),

		field.Uint32("process_time").
			Optional().
			Nillable().
			Comment("耗时(毫秒)"),

		field.Time("execute_time").
			Optional().
			Nillable().
			Comment("执行时间"),
	}
}

func (TaskLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TenantID[uint32]{},
	}
}

func (TaskLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "task_id", "execute_time").
			StorageKey("idx_sys_task_log_tenant_task_execute_time"),
		index.Fields("tenant_id", "status", "execute_time").
			StorageKey("idx_sys_task_log_tenant_status_execute_time"),
	}
}

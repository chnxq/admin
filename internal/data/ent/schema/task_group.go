package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/chnxq/x-crud/entgo/mixin"
)

// TaskGroup holds the schema definition for the TaskGroup entity.
type TaskGroup struct {
	ent.Schema
}

func (TaskGroup) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_task_groups",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("任务分组表"),
	}
}

func (TaskGroup) Fields() []ent.Field {
	return []ent.Field{
		field.String("group_name").
			NotEmpty().
			MaxLen(100).
			Comment("分组名称"),
	}
}

func (TaskGroup) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId64{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint64]{},
	}
}

func (TaskGroup) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "group_name").
			Unique().
			StorageKey("uix_sys_task_group_tenant_name"),
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_sys_task_group_tenant_created_at"),
	}
}

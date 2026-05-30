package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/chnxq/x-crud/entgo/mixin"
)

// DictLabel holds the schema definition for the DictLabel entity.
type DictLabel struct {
	ent.Schema
}

func (DictLabel) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_dict_labels",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("字典标签表"),
	}
}

func (DictLabel) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("category_id").
			Comment("所属分类 ID"),

		field.String("label_key").
			NotEmpty().
			Immutable().
			Comment("标签稳定键，如 page.user.list.title"),

		field.String("label_code").
			Optional().
			Nillable().
			Comment("标签编码/机器值，用于兼容枚举式业务"),

		field.Enum("label_kind").
			NamedValues(
				"Text", "TEXT",
				"Menu", "MENU",
				"Message", "MESSAGE",
				"Enum", "ENUM",
				"Hint", "HINT",
				"Badge", "BADGE",
			).
			Default("TEXT").
			Comment("标签类型"),

		field.String("default_text").
			Optional().
			Nillable().
			Comment("默认文本，用于缺省回退"),

		field.JSON("payload_json", map[string]any{}).
			Optional().
			Comment("扩展元数据，如 icon、color、route、template"),

		field.Bool("is_builtin").
			Default(false).
			Comment("是否系统内置"),

		field.Enum("status").
			NamedValues(
				"Off", "OFF",
				"On", "ON",
			).
			Default("ON").
			Comment("标签状态"),
	}
}

func (DictLabel) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.IsEnabled{},
		mixin.SortOrder{},
		mixin.TenantID[uint32]{},
		mixin.Remark{},
		mixin.Description{},
	}
}

func (DictLabel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("category", DictCategory.Type).
			Ref("labels").
			Unique().
			Required().
			Field("category_id"),
		edge.To("i18ns", DictLabelI18n.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}

func (DictLabel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").StorageKey("idx_dict_labels_tenant_id"),
		index.Fields("tenant_id", "category_id", "label_key").
			Unique().
			StorageKey("uix_dict_labels_tenant_category_key"),
		index.Fields("tenant_id", "label_code").
			StorageKey("idx_dict_labels_tenant_label_code"),
		index.Fields("category_id", "label_kind").
			StorageKey("idx_dict_labels_category_kind"),
		index.Fields("is_builtin").
			StorageKey("idx_dict_labels_is_builtin"),
		index.Fields("status").
			StorageKey("idx_dict_labels_status"),
	}
}

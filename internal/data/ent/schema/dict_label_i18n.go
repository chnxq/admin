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

// DictLabelI18n holds the schema definition for the DictLabelI18n entity.
type DictLabelI18n struct {
	ent.Schema
}

func (DictLabelI18n) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_dict_label_i18n",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("字典标签语言值表"),
	}
}

func (DictLabelI18n) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("label_id").
			Comment("所属标签 ID"),

		field.String("language_code").
			NotEmpty().
			Immutable().
			Comment("语言编码，如 zh-CN、en-US"),

		field.String("text_value").
			NotEmpty().
			Comment("完整语言值"),

		field.String("short_text").
			Optional().
			Nillable().
			Comment("短文本，用于卡片/按钮/徽标等紧凑场景"),
	}
}

func (DictLabelI18n) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
		mixin.Description{},
	}
}

func (DictLabelI18n) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("label", DictLabel.Type).
			Ref("i18ns").
			Unique().
			Required().
			Field("label_id"),
	}
}

func (DictLabelI18n) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("language_code").
			StorageKey("idx_dict_label_i18n_language_code"),
		index.Fields("label_id", "language_code").
			Unique().
			StorageKey("uix_dict_label_i18n_label_language"),
	}
}

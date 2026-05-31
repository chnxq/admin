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

// DictCategoryI18n holds the schema definition for the DictCategoryI18n entity.
type DictCategoryI18n struct {
	ent.Schema
}

func (DictCategoryI18n) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_dict_category_i18n",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("字典分类语言值表"),
	}
}

func (DictCategoryI18n) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("category_id").
			Comment("所属分类 ID"),

		field.String("language_code").
			NotEmpty().
			Immutable().
			Comment("语言编码，如 zh-CN、en-US"),

		field.String("display_name").
			NotEmpty().
			Comment("分类显示名"),
	}
}

func (DictCategoryI18n) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
		mixin.Description{},
	}
}

func (DictCategoryI18n) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("category", DictCategory.Type).
			Ref("i18ns").
			Unique().
			Required().
			Field("category_id"),
	}
}

func (DictCategoryI18n) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("language_code").
			StorageKey("idx_dict_category_i18n_language_code"),
		index.Fields("category_id", "language_code").
			Unique().
			StorageKey("uix_dict_category_i18n_category_language"),
	}
}

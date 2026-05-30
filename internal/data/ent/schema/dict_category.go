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

// DictCategory holds the schema definition for the DictCategory entity.
type DictCategory struct {
	ent.Schema
}

func (DictCategory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sys_dict_categories",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("字典分类表"),
	}
}

func (DictCategory) Fields() []ent.Field {
	return []ent.Field{
		field.String("category_key").
			NotEmpty().
			Comment("分类稳定键，如 page、menu、prompt、user_management"),

		field.String("category_name").
			NotEmpty().
			Comment("分类显示名，后台默认展示名称"),

		field.Enum("category_level").
			NamedValues(
				"Root", "ROOT",
				"Child", "CHILD",
			).
			Default("CHILD").
			Comment("分类层级"),

		field.Enum("scene").
			NamedValues(
				"Page", "PAGE",
				"Menu", "MENU",
				"Prompt", "PROMPT",
				"Device", "DEVICE",
				"Other", "OTHER",
			).
			Default("OTHER").
			Comment("分类场景"),

		field.Bool("is_builtin").
			Default(false).
			Comment("是否系统内置"),
	}
}

func (DictCategory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.IsEnabled{},
		mixin.SortOrder{},
		mixin.TenantID[uint32]{},
		mixin.Remark{},
		mixin.Description{},
		mixin.Tree[DictCategory]{},
		mixin.TreePath{},
	}
}

func (DictCategory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("labels", DictLabel.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
		edge.To("i18ns", DictCategoryI18n.Type).
			Annotations(entsql.Annotation{
				OnDelete: entsql.Cascade,
			}),
	}
}

func (DictCategory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id").StorageKey("idx_dict_categories_tenant_id"),
		index.Fields("tenant_id", "parent_id", "category_key").
			Unique().
			StorageKey("uix_dict_categories_tenant_parent_key"),
		index.Fields("tenant_id", "scene", "category_level").
			StorageKey("idx_dict_categories_tenant_scene_level"),
		index.Fields("parent_id").
			StorageKey("idx_dict_categories_parent_id"),
		index.Fields("is_builtin").
			StorageKey("idx_dict_categories_is_builtin"),
		index.Fields("is_enabled").
			StorageKey("idx_dict_categories_is_enabled"),
	}
}

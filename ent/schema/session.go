package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type Session struct {
	ent.Schema
}

func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.String("user_id").
			Annotations(entsql.Annotation{Size: 50}).
			Comment("用户Id"),
		field.Bool("status").
			Comment("会话是否开启").Default(false),
		field.Time("created_at").
			Default(time.Now).
			Annotations(&entsql.Annotation{
				Default: "CURRENT_TIMESTAMP",
			}).
			Immutable(),
		field.Time("updated_at").
			//SchemaType(map[string]string{
			//	dialect.MySQL: "timestamp",
			//}).
			Annotations(&entsql.Annotation{
				Default: "CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP",
			}).
			Default(time.Now).
			UpdateDefault(time.Now),
		field.Int("deleted_at").Default(0),
	}
}

func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("messages", Message.Type),
	}
}

func (Session) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "user_id"),
	}
}

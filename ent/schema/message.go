package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

type Message struct {
	ent.Schema
}

func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.Int("session_id").
			Optional().
			Comment("会话Id"),
		field.String("from_user_id").
			Annotations(entsql.Annotation{Size: 50}).
			Comment("消息发送者Id"),
		field.String("to_user_id").
			Annotations(entsql.Annotation{Size: 50}).
			Comment("消息接收者Id"),
		field.Text("content").
			Comment("消息内容"),
		field.Int("spouse_id").
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Annotations(&entsql.Annotation{
				Default: "CURRENT_TIMESTAMP",
			}).
			Immutable(),
	}
}

func (Message) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("spouse", Message.Type).
			Unique().
			Field("spouse_id"),
		edge.From("session", Session.Type).
			Ref("messages").
			Field("session_id").
			Unique(),
	}
}

func (Message) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_id", "from_user_id", "created_at"),
		index.Fields("session_id", "to_user_id", "created_at"),
	}
}

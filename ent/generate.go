package ent

//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate --feature  sql/lock,sql/upsert,privacy,entql,schema/snapshot,sql/modifier,sql/execquery ./schema --target ./chatent

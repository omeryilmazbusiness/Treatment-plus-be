// Package ctxkey holds typed context values. clinic_id is never read from body/query.
package ctxkey

import "context"

type key int

const (
	keyTraceID key = iota + 1
	keyClinicID
	keyUserID
	keyRole
)

func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyTraceID, id)
}

func TraceID(ctx context.Context) string {
	v, _ := ctx.Value(keyTraceID).(string)
	return v
}

func WithClinicID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyClinicID, id)
}

func ClinicID(ctx context.Context) string {
	v, _ := ctx.Value(keyClinicID).(string)
	return v
}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyUserID, id)
}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(keyUserID).(string)
	return v
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, keyRole, role)
}

func Role(ctx context.Context) string {
	v, _ := ctx.Value(keyRole).(string)
	return v
}

// Package connectionowner carries the account whose credentials a task may use.
package connectionowner

import "context"

type key struct{}

// WithUser scopes credential access to a user, including background tasks.
func WithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, key{}, userID)
}

// UserID returns the owner; an empty owner never matches a signed-in account.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(key{}).(string)
	return id
}

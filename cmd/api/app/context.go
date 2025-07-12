package app

import (
	"context"
	"net/http"

	"github.com/OpenConnectOUSL/backend-api-v1/internal/data"
)

type contextKey string

const userContextKey = contextKey("user")

// ContextSetUser adds a user to the request context
func (app *Application) ContextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// ContextGetUser retrieves a user from the request context
func (app *Application) ContextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}

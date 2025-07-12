package routes

import (
	"net/http"

	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/app"
	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/handlers"
	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/middleware"
	"github.com/julienschmidt/httprouter"
)

// Setup configures and returns the application routes
func Setup(app *app.Application) http.Handler {
	router := httprouter.New()

	// Set custom error handlers
	router.NotFound = http.HandlerFunc(app.NotFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.MethodNotAllowedResponse)

	// Health check route
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", handlers.Healthcheck(app))

	// Ideas routes
	router.HandlerFunc(http.MethodGet, "/v1/ideas", middleware.RequirePermission(app, "ideas:read")(handlers.ListIdeas(app)))
	router.HandlerFunc(http.MethodPost, "/v1/ideas", middleware.RequirePermission(app, "ideas:write")(handlers.CreateIdea(app)))
	router.HandlerFunc(http.MethodGet, "/v1/ideas/:id", middleware.RequirePermission(app, "ideas:read")(handlers.ShowIdea(app)))
	router.HandlerFunc(http.MethodPatch, "/v1/ideas/:id", middleware.RequirePermission(app, "ideas:write")(handlers.UpdateIdea(app)))
	router.HandlerFunc(http.MethodDelete, "/v1/ideas/:id", middleware.RequirePermission(app, "ideas:write")(handlers.DeleteIdea(app)))

	// User routes
	router.HandlerFunc(http.MethodPost, "/v1/users", handlers.RegisterUser(app))
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", handlers.ActivateUser(app))
	router.HandlerFunc(http.MethodPut, "/v1/users/password-reset", handlers.UpdateUserPassword(app))

	// Authentication token routes
	router.HandlerFunc(http.MethodPost, "/v1/auth/tokens/authentication", handlers.CreateAuthenticationToken(app))
	router.HandlerFunc(http.MethodPost, "/v1/auth/tokens/password-reset-request", handlers.CreatePasswordResetToken(app))

	// OAuth routes
	router.HandlerFunc(http.MethodGet, "/v1/auth/google/login", handlers.GoogleLogin(app))
	router.HandlerFunc(http.MethodGet, "/v1/auth/google/callback", handlers.GoogleCallback(app))

	// User profile routes
	router.HandlerFunc(http.MethodPost, "/v1/user-profiles", middleware.RequireActivatedUser(app)(handlers.CreateUserProfile(app)))
	router.HandlerFunc(http.MethodGet, "/v1/user-profiles/:id", middleware.RequireAuthenticatedUser(app)(handlers.GetUserProfile(app)))
	router.HandlerFunc(http.MethodPatch, "/v1/user-profiles/:id", middleware.RequireActivatedUser(app)(handlers.UpdateUserProfile(app)))

	// File upload routes
	router.HandlerFunc(http.MethodPost, "/v1/files/upload", middleware.RequireAuthenticatedUser(app)(handlers.ServePDFHandler(app)))

	// Apply middleware chain
	return middleware.EnableCORS(app)(
		middleware.RecoverPanic(app)(
			middleware.RateLimit(app)(
				middleware.Authenticate(app)(router))))
}

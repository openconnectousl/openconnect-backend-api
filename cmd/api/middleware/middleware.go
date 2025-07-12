package middleware

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/app"
	"github.com/OpenConnectOUSL/backend-api-v1/internal/data"
	"github.com/OpenConnectOUSL/backend-api-v1/internal/validator"
	"golang.org/x/time/rate"
)

// RecoverPanic recovers from panics and returns a 500 error
func RecoverPanic(appPtr *app.Application) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.Header().Set("Connection", "close")
					appPtr.ServerErrorResponse(w, r, fmt.Errorf("%s", err))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit implements rate limiting middleware
func RateLimit(appPtr *app.Application) func(http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !appPtr.Config.Limiter.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				appPtr.ServerErrorResponse(w, r, err)
				return
			}

			mu.Lock()

			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(appPtr.Config.Limiter.RPS), appPtr.Config.Limiter.Burst),
				}
			}

			clients[ip].lastSeen = time.Now()

			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				appPtr.RateLimitExceededResponse(w, r)
				return
			}
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// Authenticate validates the authentication token and sets the user in context
func Authenticate(appPtr *app.Application) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Vary", "Authorization")

			authorizationHeader := r.Header.Get("Authorization")

			if authorizationHeader == "" {
				r = appPtr.ContextSetUser(r, data.AnonymousUser)
				next.ServeHTTP(w, r)
				return
			}

			headerParts := strings.Split(authorizationHeader, " ")
			if len(headerParts) != 2 || headerParts[0] != "Bearer" {
				appPtr.InvalidAuthenticationTokenResponse(w, r)
				return
			}

			token := headerParts[1]

			v := validator.New()

			if data.ValidateTokenPlaintext(v, token); !v.Valid() {
				appPtr.InvalidCredentialsResponse(w, r)
				return
			}

			user, err := appPtr.Models.Users.GetForToken(data.ScopeAuthentication, token)
			if err != nil {
				switch {
				case errors.Is(err, data.ErrRecordNotFound):
					appPtr.InvalidAuthenticationTokenResponse(w, r)
				default:
					appPtr.ServerErrorResponse(w, r, err)
				}
				return
			}

			r = appPtr.ContextSetUser(r, user)

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuthenticatedUser requires that the user is authenticated
func RequireAuthenticatedUser(appPtr *app.Application) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := appPtr.ContextGetUser(r)

			if user.IsAnonymous() {
				appPtr.AuthenticationRequiredResponse(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireActivatedUser requires that the user is activated
func RequireActivatedUser(appPtr *app.Application) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := appPtr.ContextGetUser(r)

			if !user.Activated {
				appPtr.ErrorResponse(w, r, http.StatusForbidden, "user account not activated")
				return
			}
			next.ServeHTTP(w, r)
		})

		return RequireAuthenticatedUser(appPtr)(fn)
	}
}

// RequirePermission requires that the user has the specified permission
func RequirePermission(appPtr *app.Application, code string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		fn := func(w http.ResponseWriter, r *http.Request) {
			user := appPtr.ContextGetUser(r)

			permissions, err := appPtr.Models.Permissions.GetAllForUser(user.ID)
			if err != nil {
				appPtr.ServerErrorResponse(w, r, err)
				return
			}

			if !permissions.Include(code) {
				appPtr.NotPermittedResponse(w, r)
				return
			}

			next.ServeHTTP(w, r)
		}

		return RequireActivatedUser(appPtr)(fn)
	}
}

// EnableCORS enables CORS for trusted origins
func EnableCORS(appPtr *app.Application) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Vary", "Origin")
			w.Header().Add("Vary", "Access-Control-Request-Method")

			origin := r.Header.Get("Origin")

			if origin != "" {
				for i := range appPtr.Config.CORS.TrustedOrigins {
					if origin == appPtr.Config.CORS.TrustedOrigins[i] {
						w.Header().Set("Access-Control-Allow-Origin", origin)

						if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
							w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
							w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
							w.WriteHeader(http.StatusOK)
							return
						}
						break
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

package handlers

import (
	"net/http"

	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/app"
)

// Healthcheck returns a healthcheck handler
func Healthcheck(appPtr *app.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		env := map[string]any{
			"status": "available",
			"system_info": map[string]string{
				"environment": appPtr.Config.Env,
				"version":     app.Version,
			},
		}
		err := appPtr.WriteJSON(w, http.StatusOK, env, nil)
		if err != nil {
			appPtr.ServerErrorResponse(w, r, err)
		}
	}
}

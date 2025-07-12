package app

import (
	"sync"

	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/config"
	"github.com/OpenConnectOUSL/backend-api-v1/internal/data"
	"github.com/OpenConnectOUSL/backend-api-v1/internal/jsonlog"
	"github.com/OpenConnectOUSL/backend-api-v1/internal/mailer"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const Version = "1.0.0"

// Application holds the dependencies for our HTTP handlers, helpers, and middleware
type Application struct {
	Config            *config.Config
	Logger            *jsonlog.Logger
	Models            data.Models
	Mailer            mailer.Mailer
	WG                sync.WaitGroup
	GoogleOauthConfig *oauth2.Config
}

// InitGoogleOAuth initializes the Google OAuth configuration
func (app *Application) InitGoogleOAuth() {
	app.GoogleOauthConfig = &oauth2.Config{
		ClientID:     app.Config.OAuth.GoogleClientID,
		ClientSecret: app.Config.OAuth.GoogleClientSecret,
		RedirectURL:  app.Config.OAuth.RedirectURI,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

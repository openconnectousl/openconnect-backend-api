package main

import (
	"os"

	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/app"
	appconfig "github.com/OpenConnectOUSL/backend-api-v1/cmd/api/config"
	"github.com/OpenConnectOUSL/backend-api-v1/cmd/api/server"
	"github.com/OpenConnectOUSL/backend-api-v1/internal/data"
	"github.com/OpenConnectOUSL/backend-api-v1/internal/jsonlog"
	"github.com/OpenConnectOUSL/backend-api-v1/internal/mailer"
)

func main() {
	// Load configuration
	cfg, err := appconfig.LoadConfig()
	if err != nil {
		panic(err)
	}

	// Initialize logger
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)
	if logger == nil {
		panic("Logger is not initialized")
	}

	// Open database connection
	db, err := cfg.OpenDB()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()

	logger.PrintInfo("database connection pool established", nil)

	// Initialize application
	appPtr := &app.Application{
		Config: cfg,
		Logger: logger,
		Models: data.NewModels(db),
		Mailer: mailer.New(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.Sender),
	}

	// Initialize Google OAuth
	appPtr.InitGoogleOAuth()

	// Start server
	err = server.Serve(appPtr)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}

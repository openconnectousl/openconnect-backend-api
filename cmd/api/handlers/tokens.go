package handlers

import (
    "errors"
    "fmt"
    "net/http"
    "time"

    "github.com/OpenConnectOUSL/backend-api-v1/cmd/api/app"
    "github.com/OpenConnectOUSL/backend-api-v1/internal/data"
    "github.com/OpenConnectOUSL/backend-api-v1/internal/validator"
)

// CreateAuthenticationToken creates an authentication token for login
func CreateAuthenticationToken(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var input struct {
            Email    string `json:"email"`
            Password string `json:"password"`
        }

        err := appPtr.ReadJSON(w, r, &input)
        if err != nil {
            appPtr.BadRequestResponse(w, r, err)
            return
        }

        v := validator.New()
        data.ValidateEmail(v, input.Email)
        data.ValidatePasswordPlaintext(v, input.Password)

        if !v.Valid() {
            appPtr.FailedValidationResponse(w, r, v.Errors)
            return
        }

        user, err := appPtr.Models.Users.GetByEmail(input.Email)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrRecordNotFound):
                appPtr.InvalidCredentialsResponse(w, r)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        match, err := user.Password.Matches(input.Password)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        if !match {
            appPtr.InvalidCredentialsResponse(w, r)
            return
        }

        token, err := appPtr.Models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        err = appPtr.WriteJSON(w, http.StatusCreated, app.Envelope{"authentication_token": token}, nil)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }
    }
}

// CreatePasswordResetToken creates a password reset token
func CreatePasswordResetToken(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var input struct {
            Email string `json:"email"`
        }

        err := appPtr.ReadJSON(w, r, &input)
        if err != nil {
            appPtr.BadRequestResponse(w, r, err)
            return
        }

        v := validator.New()

        if data.ValidateEmail(v, input.Email); !v.Valid() {
            appPtr.FailedValidationResponse(w, r, v.Errors)
            return
        }

        user, err := appPtr.Models.Users.GetByEmail(input.Email)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrRecordNotFound):
                v.AddError("email", "no user found with this email address")
                appPtr.FailedValidationResponse(w, r, v.Errors)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        if !user.Activated {
            v.AddError("email", "user account is not activated")
            appPtr.FailedValidationResponse(w, r, v.Errors)
            return
        }

        token, err := appPtr.Models.Tokens.New(user.ID, 45*time.Minute, data.ScopePasswordReset)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        appPtr.WG.Add(1)
        go func() {
            defer appPtr.WG.Done()
            emailData := map[string]any{
                "passwordResetToken": token.Plaintext,
                "frontendURL":        appPtr.Config.FrontendURL,
            }
            fmt.Println(emailData)
            err = appPtr.Mailer.Send(user.Email, "password_reset", emailData)
            if err != nil {
                appPtr.Logger.PrintError(err, nil)
            }
        }()

        env := app.Envelope{"message": "an email will be sent to you containing password reset instructions"}

        err = appPtr.WriteJSON(w, http.StatusAccepted, env, nil)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }
    }
}
package handlers

import (
    "errors"
    "net/http"
    "time"

    "github.com/OpenConnectOUSL/backend-api-v1/cmd/api/app"
    "github.com/OpenConnectOUSL/backend-api-v1/internal/data"
    "github.com/OpenConnectOUSL/backend-api-v1/internal/validator"
)

// RegisterUser creates a new user account
func RegisterUser(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var input struct {
            UserName string `json:"username"`
            Email    string `json:"email"`
            Password string `json:"password"`
        }

        err := appPtr.ReadJSON(w, r, &input)
        if err != nil {
            appPtr.BadRequestResponse(w, r, err)
            return
        }

        _, err = appPtr.Models.Users.GetByEmail(input.Email)
        if err == nil {
            appPtr.FailedValidationResponse(w, r, map[string]string{"email": "a user with this email address already exists"})
            return
        } else if !errors.Is(err, data.ErrRecordNotFound) {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        user := &data.User{
            UserName:          input.UserName,
            Email:             input.Email,
            UserType:          "normal",
            Activated:         false,
            HasProfileCreated: false,
        }

        err = user.Password.Set(input.Password)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        v := validator.New()

        if data.ValidateUser(v, user); !v.Valid() {
            appPtr.FailedValidationResponse(w, r, v.Errors)
            return
        }

        err = appPtr.Models.Users.Insert(user)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrDuplicateEmail):
                v.AddError("email", "a user with this email address already exists")
                appPtr.FailedValidationResponse(w, r, v.Errors)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        err = appPtr.Models.Permissions.AddForUser(user.ID, "ideas:read")
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        token, err := appPtr.Models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        appPtr.WG.Add(1)
        go func() {
            defer appPtr.WG.Done()
            emailData := map[string]any{
                "activationToken": token.Plaintext,
                "userName":        user.UserName,
                "frontendURL":     appPtr.Config.FrontendURL,
            }
            err = appPtr.Mailer.Send(user.Email, "user_welcome", emailData)
            if err != nil {
                appPtr.Logger.PrintError(err, nil)
            }
        }()

        err = appPtr.WriteJSON(w, http.StatusAccepted, app.Envelope{"user": user}, nil)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }
    }
}

// ActivateUser activates a user account with a token
func ActivateUser(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var input struct {
            TokenPlainText string `json:"token"`
        }

        err := appPtr.ReadJSON(w, r, &input)
        if err != nil {
            appPtr.BadRequestResponse(w, r, err)
            return
        }

        v := validator.New()

        if data.ValidateTokenPlaintext(v, input.TokenPlainText); !v.Valid() {
            appPtr.FailedValidationResponse(w, r, v.Errors)
            return
        }

        user, err := appPtr.Models.Users.GetForToken(data.ScopeActivation, input.TokenPlainText)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrRecordNotFound):
                v.AddError("token", "invalid or expired activation token")
                appPtr.FailedValidationResponse(w, r, v.Errors)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        user.Activated = true

        err = appPtr.Models.Users.Update(user)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrEditConflict):
                appPtr.EditConflictResponse(w, r)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        err = appPtr.Models.Permissions.AddForUser(user.ID, "ideas:write")
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        err = appPtr.Models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        err = appPtr.WriteJSON(w, http.StatusOK, app.Envelope{"user": user}, nil)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }
    }
}

// UpdateUserPassword updates a user's password using a reset token
func UpdateUserPassword(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var input struct {
            Password       string `json:"password"`
            TokenPlaintext string `json:"token"`
        }

        err := appPtr.ReadJSON(w, r, &input)
        if err != nil {
            appPtr.BadRequestResponse(w, r, err)
            return
        }

        v := validator.New()

        data.ValidatePasswordPlaintext(v, input.Password)
        data.ValidateTokenPlaintext(v, input.TokenPlaintext)

        if !v.Valid() {
            appPtr.FailedValidationResponse(w, r, v.Errors)
            return
        }

        user, err := appPtr.Models.Users.GetForToken(data.ScopePasswordReset, input.TokenPlaintext)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrRecordNotFound):
                v.AddError("token", "invalid or expired password reset token")
                appPtr.FailedValidationResponse(w, r, v.Errors)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        err = user.Password.Set(input.Password)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        err = appPtr.Models.Users.Update(user)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrEditConflict):
                appPtr.EditConflictResponse(w, r)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        err = appPtr.Models.Tokens.DeleteAllForUser(data.ScopePasswordReset, user.ID)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        env := app.Envelope{"message": "your password was successfully reset"}

        err = appPtr.WriteJSON(w, http.StatusOK, env, nil)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }
    }
}
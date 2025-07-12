package handlers

import (
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "time"

    "github.com/OpenConnectOUSL/backend-api-v1/cmd/api/app"
    "github.com/OpenConnectOUSL/backend-api-v1/internal/data"
)

// GoogleLogin initiates Google OAuth login
func GoogleLogin(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        state := generateStateOauthCookie(w)

        appPtr.Logger.PrintInfo("Setting oauthState cookie", map[string]string{
            "state": state,
        })
        url := appPtr.GoogleOauthConfig.AuthCodeURL(state)
        http.Redirect(w, r, url, http.StatusTemporaryRedirect)
    }
}

// GoogleCallback handles Google OAuth callback
func GoogleCallback(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        state := r.URL.Query().Get("state")
        code := r.URL.Query().Get("code")

        // Verify state token
        cookie, err := r.Cookie("oauthstate")
        if err != nil {
            appPtr.Logger.PrintInfo("Missing cookie", map[string]string{
                "received_state": state,
                "error":          err.Error(),
            })
            appPtr.InvalidCredentialsResponse(w, r)
            return
        }

        if cookie.Value != state {
            appPtr.Logger.PrintInfo("State token mismatch", map[string]string{
                "received_state": state,
                "cookie_state":   cookie.Value,
            })
            appPtr.InvalidCredentialsResponse(w, r)
            return
        }

        // Exchange code for token
        token, err := appPtr.GoogleOauthConfig.Exchange(r.Context(), code)
        if err != nil {
            appPtr.Logger.PrintError(err, map[string]string{"message": "Token exchange failed"})
            appPtr.BadRequestResponse(w, r, err)
            return
        }

        // Get user info from Google
        googleUser, err := getGoogleUserInfo(token.AccessToken)
        if err != nil {
            appPtr.Logger.PrintError(err, map[string]string{"message": "Failed to get Google user info"})
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        existingUser, err := appPtr.Models.Users.GetByEmail(googleUser.Email)
        if err == nil {
            authToken, err := appPtr.Models.Tokens.New(existingUser.ID, 24*time.Hour, data.ScopeAuthentication)
            if err != nil {
                appPtr.Logger.PrintError(err, map[string]string{"message": "Failed to generate authentication token"})
                appPtr.ServerErrorResponse(w, r, err)
                return
            }

            redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", appPtr.Config.FrontendURL, authToken.Plaintext)
            appPtr.Logger.PrintInfo("Redirecting existing user to frontend", map[string]string{
                "email": existingUser.Email,
                "url":   redirectURL,
            })
            http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
            return
        } else if !errors.Is(err, data.ErrRecordNotFound) {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        // Find or create user
        user, err := appPtr.Models.Users.FindOrCreateFromGoogle(googleUser)
        if err != nil {
            appPtr.Logger.PrintError(err, map[string]string{"message": "Failed to find or create user"})
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        err = appPtr.Models.Permissions.AddForUser(user.ID, "ideas:write")
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        // Generate authentication token
        authToken, err := appPtr.Models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
        if err != nil {
            appPtr.Logger.PrintError(err, map[string]string{"message": "Failed to generate authentication token"})
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", appPtr.Config.FrontendURL, authToken.Plaintext)
        appPtr.Logger.PrintInfo("Redirecting to frontend", map[string]string{
            "url": redirectURL,
        })
        http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
    }
}

func generateStateOauthCookie(w http.ResponseWriter) string {
    expiration := time.Now().Add(365 * 24 * time.Hour)
    b := make([]byte, 16)
    rand.Read(b)
    state := base64.URLEncoding.EncodeToString(b)
    cookie := http.Cookie{
        Name:     "oauthstate",
        Value:    state,
        Expires:  expiration,
        HttpOnly: true,
    }
    http.SetCookie(w, &cookie)
    return state
}

func getGoogleUserInfo(accessToken string) (*data.GoogleUser, error) {
    resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var user data.GoogleUser
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        return nil, err
    }

    return &user, nil
}
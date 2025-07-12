package handlers

import (
    "errors"
    "fmt"
    "net/http"

    "github.com/OpenConnectOUSL/backend-api-v1/cmd/api/app"
    "github.com/OpenConnectOUSL/backend-api-v1/internal/data"
    "github.com/OpenConnectOUSL/backend-api-v1/internal/validator"
    "github.com/google/uuid"
)

// CreateIdea creates a new idea
func CreateIdea(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var input struct {
            Title            string   `json:"title"`
            Description      string   `json:"description"`
            PDF              string   `json:"pdf"`
            Category         string   `json:"category"`
            Tags             []string `json:"tags"`
            UserID           string   `json:"user_id"`
            LearningOutcome  string   `json:"learning_outcome"`
            RecommendedLevel string   `json:"recommended_level"`
            GitHubLink       string   `json:"github_link"`
            WebsiteLink      string   `json:"website_link"`
        }

        err := appPtr.ReadJSON(w, r, &input)
        if err != nil {
            appPtr.BadRequestResponse(w, r, err)
            return
        }

        user := appPtr.ContextGetUser(r)

        var userID uuid.UUID

        if input.UserID != "" {
            if len(input.UserID) != 36 {
                appPtr.BadRequestResponse(w, r, fmt.Errorf("user_id must be a valid UUID of length 36"))
                return
            }
            userID, err = uuid.Parse(input.UserID)
            if err != nil {
                appPtr.BadRequestResponse(w, r, fmt.Errorf("user_id must be a valid UUID"))
                return
            }
        } else {
            userID = user.ID
        }

        // Get existing profile
        profile, err := appPtr.Models.UserProfile.GetByUserID(user.ID)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrRecordNotFound):
                appPtr.NotFoundResponse(w, r)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }
        fmt.Println("Profile ID:", profile)

        pdfID, err := appPtr.ProcessAndSavePDF(input.PDF, w, r)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        idea := &data.Idea{
            Title:            input.Title,
            Description:      input.Description,
            Category:         input.Category,
            Tags:             input.Tags,
            UserID:           userID,
            IdeaSourceID:     pdfID,
            LearningOutcome:  input.LearningOutcome,
            RecommendedLevel: input.RecommendedLevel,
            GitHubLink:       input.GitHubLink,
            WebsiteLink:      input.WebsiteLink,
        }

        v := validator.New()

        if data.ValidateIdea(v, idea); !v.Valid() {
            appPtr.FailedValidationResponse(w, r, v.Errors)
            return
        }

        err = appPtr.Models.Ideas.Insert(idea)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        headers := make(http.Header)
        headers.Set("Location", fmt.Sprintf("/v1/ideas/%d", idea.ID))

        err = appPtr.WriteJSON(w, http.StatusCreated, app.Envelope{"idea": idea}, headers)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }

        fmt.Fprintf(w, "%v\n", "Idea submitted successfully")
    }
}

// ShowIdea retrieves a specific idea by ID
func ShowIdea(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id, err := appPtr.ReadIDParam(r)
        if err != nil {
            appPtr.NotFoundResponse(w, r)
            return
        }

        idea, err := appPtr.Models.Ideas.Get(id)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrRecordNotFound):
                appPtr.NotFoundResponse(w, r)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        err = appPtr.WriteJSON(w, http.StatusOK, app.Envelope{"idea": idea}, nil)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }
    }
}

// UpdateIdea updates an existing idea
func UpdateIdea(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id, err := appPtr.ReadIDParam(r)
        if err != nil {
            appPtr.NotFoundResponse(w, r)
            return
        }

        idea, err := appPtr.Models.Ideas.Get(id)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrRecordNotFound):
                appPtr.NotFoundResponse(w, r)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        var input struct {
            Title            *string  `json:"title"`
            Description      *string  `json:"description"`
            Category         *string  `json:"category"`
            Tags             []string `json:"tags"`
            PdfBase64        *string  `json:"pdfBase64"`
            LearningOutcome  *string  `json:"learning_outcome"`
            RecommendedLevel *string  `json:"recommended_level"`
            GitHubLink       *string  `json:"github_link"`
            WebsiteLink      *string  `json:"website_link"`
            Status           *string  `json:"status"`
        }

        err = appPtr.ReadJSON(w, r, &input)
        if err != nil {
            appPtr.BadRequestResponse(w, r, err)
            return
        }

        // Only process PDF if one is provided
        var uniqueID string
        if input.PdfBase64 != nil && *input.PdfBase64 != "" {
            uniqueID, err = appPtr.ProcessAndSavePDF(*input.PdfBase64, w, r)
            if err != nil {
                appPtr.ServerErrorResponse(w, r, err)
                return
            }
        }

        if input.Title != nil {
            idea.Title = *input.Title
        }

        if input.Description != nil {
            idea.Description = *input.Description
        }

        if input.Category != nil {
            idea.Category = *input.Category
        }

        if input.Tags != nil {
            idea.Tags = input.Tags
        }

        if input.PdfBase64 != nil && uniqueID != "" {
            idea.Pdf = uniqueID
        }

        if input.LearningOutcome != nil {
            idea.LearningOutcome = *input.LearningOutcome
        }

        if input.RecommendedLevel != nil {
            idea.RecommendedLevel = *input.RecommendedLevel
        }

        if input.GitHubLink != nil {
            idea.GitHubLink = *input.GitHubLink
        }

        if input.WebsiteLink != nil {
            idea.WebsiteLink = *input.WebsiteLink
        }

        v := validator.New()

        if data.ValidateIdea(v, idea); !v.Valid() {
            appPtr.FailedValidationResponse(w, r, v.Errors)
            return
        }

        err = appPtr.Models.Ideas.Update(idea)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrEditConflict):
                appPtr.EditConflictResponse(w, r)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        err = appPtr.WriteJSON(w, http.StatusOK, app.Envelope{"idea": idea}, nil)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }
    }
}

// DeleteIdea deletes an idea by ID
func DeleteIdea(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id, err := appPtr.ReadIDParam(r)
        if err != nil {
            appPtr.NotFoundResponse(w, r)
            return
        }

        err = appPtr.Models.Ideas.Delete(id)
        if err != nil {
            switch {
            case errors.Is(err, data.ErrRecordNotFound):
                appPtr.NotFoundResponse(w, r)
            default:
                appPtr.ServerErrorResponse(w, r, err)
            }
            return
        }

        err = appPtr.WriteJSON(w, http.StatusOK, app.Envelope{"message": "idea deleted successfully"}, nil)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }
    }
}

// ListIdeas lists ideas with filtering and pagination
func ListIdeas(appPtr *app.Application) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var input struct {
            Title    string
            Category string
            Tags     []string
            data.Filters
        }

        v := validator.New()

        qs := r.URL.Query()

        input.Title = appPtr.ReadString(qs, "title", "")
        input.Category = appPtr.ReadString(qs, "category", "")
        input.Tags = appPtr.ReadCSV(qs, "tags", []string{})

        input.Filters.Page = appPtr.ReadInt(qs, "page", 1, v)
        input.Filters.PageSize = appPtr.ReadInt(qs, "page_size", 20, v)
        input.Filters.Sort = appPtr.ReadString(qs, "sort", "id")
        input.Filters.SortSafelist = []string{"id", "title", "category", "-id", "-title", "-category"}

        if data.ValidateFilters(v, input.Filters); !v.Valid() {
            appPtr.FailedValidationResponse(w, r, v.Errors)
            return
        }

        ideas, metadata, err := appPtr.Models.Ideas.GetAllIdeas(input.Title, input.Tags, input.Filters)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
            return
        }

        err = appPtr.WriteJSON(w, http.StatusOK, app.Envelope{"metadata": metadata, "ideas": ideas}, nil)
        if err != nil {
            appPtr.ServerErrorResponse(w, r, err)
        }
    }
}
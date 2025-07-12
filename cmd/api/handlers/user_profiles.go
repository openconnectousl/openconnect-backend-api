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

// CreateUserProfile creates a new user profile
func CreateUserProfile(appPtr *app.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := appPtr.ContextGetUser(r)

		_, err := appPtr.Models.UserProfile.GetByUserID(user.ID)
		if err == nil {
			appPtr.FailedValidationResponse(w, r, map[string]string{"profile": "profile already exists for this user"})
			return
		} else if !errors.Is(err, data.ErrRecordNotFound) {
			appPtr.ServerErrorResponse(w, r, err)
			return
		}

		var input struct {
			Firstname string   `json:"firstname"`
			Lastname  string   `json:"lastname"`
			Avatar    string   `json:"avatar"`
			Title     string   `json:"title"`
			Bio       string   `json:"bio"`
			Faculty   string   `json:"faculty"`
			Program   string   `json:"program"`
			Degree    string   `json:"degree"`
			Year      string   `json:"year"`
			Uni       string   `json:"uni"`
			Mobile    string   `json:"mobile"`
			LinkedIn  string   `json:"linkedin"`
			GitHub    string   `json:"github"`
			FB        string   `json:"fb"`
			Skills    []string `json:"skills"`
		}

		err = appPtr.ReadJSON(w, r, &input)
		if err != nil {
			appPtr.BadRequestResponse(w, r, err)
			return
		}

		avatarID, err := appPtr.ProcessAndSaveAvatar(input.Avatar, w, r)
		if err != nil {
			return
		}

		profile := &data.Profile{
			UserID:    user.ID,
			Firstname: input.Firstname,
			Lastname:  input.Lastname,
			Avatar:    avatarID,
			Title:     input.Title,
			Bio:       input.Bio,
			Faculty:   input.Faculty,
			Program:   input.Program,
			Degree:    input.Degree,
			Year:      input.Year,
			Uni:       input.Uni,
			Mobile:    input.Mobile,
			LinkedIn:  input.LinkedIn,
			GitHub:    input.GitHub,
			FB:        input.FB,
			Skills:    input.Skills,
		}

		// Validate the profile data
		v := validator.New()
		validateProfile(v, profile)
		if !v.Valid() {
			appPtr.FailedValidationResponse(w, r, v.Errors)
			return
		}

		// Insert the profile
		err = appPtr.Models.UserProfile.Insert(profile)
		if err != nil {
			appPtr.ServerErrorResponse(w, r, err)
			return
		}

		user.HasProfileCreated = true
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

		// Return the created profile
		err = appPtr.WriteJSON(w, http.StatusCreated, app.Envelope{"profile": profile}, nil)
		if err != nil {
			appPtr.ServerErrorResponse(w, r, err)
		}
	}
}

// GetUserProfile retrieves a user profile
func GetUserProfile(appPtr *app.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := appPtr.ReadIDParam(r)
		if err != nil {
			appPtr.NotFoundResponse(w, r)
			return
		}

		profile, err := appPtr.Models.UserProfile.GetFullProfile(id)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				appPtr.NotFoundResponse(w, r)
			default:
				appPtr.ServerErrorResponse(w, r, err)
			}
			return
		}

		err = appPtr.WriteJSON(w, http.StatusOK, app.Envelope{"profile": profile}, nil)
		if err != nil {
			appPtr.ServerErrorResponse(w, r, err)
		}
	}
}

// UpdateUserProfile updates an existing user profile
func UpdateUserProfile(appPtr *app.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := appPtr.ContextGetUser(r)

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

		var input struct {
			Firstname *string  `json:"firstname"`
			Lastname  *string  `json:"lastname"`
			Avatar    *string  `json:"avatar"`
			Title     *string  `json:"title"`
			Bio       *string  `json:"bio"`
			Faculty   *string  `json:"faculty"`
			Program   *string  `json:"program"`
			Degree    *string  `json:"degree"`
			Year      *string  `json:"year"`
			Uni       *string  `json:"uni"`
			Mobile    *string  `json:"mobile"`
			LinkedIn  *string  `json:"linkedin"`
			GitHub    *string  `json:"github"`
			FB        *string  `json:"fb"`
			Skills    []string `json:"skills"`
		}

		err = appPtr.ReadJSON(w, r, &input)
		if err != nil {
			appPtr.BadRequestResponse(w, r, err)
			return
		}

		// Update profile fields if provided
		if input.Firstname != nil {
			profile.Firstname = *input.Firstname
		}
		if input.Lastname != nil {
			profile.Lastname = *input.Lastname
		}
		if input.Avatar != nil && *input.Avatar != "" {
			avatarID, err := appPtr.ProcessAndSaveAvatar(*input.Avatar, w, r)
			if err != nil {
				return
			}
			profile.Avatar = avatarID
		}
		if input.Title != nil {
			profile.Title = *input.Title
		}
		if input.Bio != nil {
			profile.Bio = *input.Bio
		}
		if input.Faculty != nil {
			profile.Faculty = *input.Faculty
		}
		if input.Program != nil {
			profile.Program = *input.Program
		}
		if input.Degree != nil {
			profile.Degree = *input.Degree
		}
		if input.Year != nil {
			profile.Year = *input.Year
		}
		if input.Uni != nil {
			profile.Uni = *input.Uni
		}
		if input.Mobile != nil {
			profile.Mobile = *input.Mobile
		}
		if input.LinkedIn != nil {
			profile.LinkedIn = *input.LinkedIn
		}
		if input.GitHub != nil {
			profile.GitHub = *input.GitHub
		}
		if input.FB != nil {
			profile.FB = *input.FB
		}
		if input.Skills != nil {
			profile.Skills = input.Skills
		}

		v := validator.New()
		validateProfile(v, profile)
		if !v.Valid() {
			appPtr.FailedValidationResponse(w, r, v.Errors)
			return
		}

		err = appPtr.Models.UserProfile.Update(profile)
		if err != nil {
			appPtr.ServerErrorResponse(w, r, err)
			return
		}

		err = appPtr.WriteJSON(w, http.StatusOK, app.Envelope{"profile": profile}, nil)
		if err != nil {
			appPtr.ServerErrorResponse(w, r, err)
		}
	}
}

// Helper function to validate profile data
func validateProfile(v *validator.Validator, profile *data.Profile) {
	v.Check(profile.UserID != uuid.Nil, "user_id", "must be provided")

	if profile.Firstname != "" {
		v.Check(len(profile.Firstname) <= 100, "firstname", "must not exceed 100 characters")
	}

	if profile.Lastname != "" {
		v.Check(len(profile.Lastname) <= 100, "lastname", "must not exceed 100 characters")
	}

	if profile.Title != "" {
		v.Check(len(profile.Title) <= 100, "title", "must not exceed 100 characters")
	}

	if profile.Bio != "" {
		v.Check(len(profile.Bio) <= 1000, "bio", "must not exceed 1000 characters")
	}

	if len(profile.Skills) > 0 {
		v.Check(len(profile.Skills) <= 20, "skills", "cannot have more than 20 skills")

		for i, skill := range profile.Skills {
			v.Check(len(skill) <= 50, fmt.Sprintf("skills.%d", i), "must not exceed 50 characters")
		}
	}
}

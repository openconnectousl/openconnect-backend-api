package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/OpenConnectOUSL/backend-api-v1/internal/validator"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Idea struct {
	ID               uuid.UUID `json:"id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	UserID           uuid.UUID `json:"user_id"`
	IdeaSourceID     string    `json:"idea_source_id,omitempty"`
	Pdf              string    `json:"pdf"`
	Category         string    `json:"category"`
	Tags             []string  `json:"tags"`
	Status           string    `json:"status"`
	Feedback         sql.NullString    `json:"-"`
	FeedbackStr      string         `json:"feedback,omitempty"` // For JSON output
	LearningOutcome  string    `json:"learning_outcome,omitempty"`
	RecommendedLevel string    `json:"recommended_level,omitempty"`
	GitHubLink       string    `json:"github_link,omitempty"`
	WebsiteLink      string    `json:"website_link,omitempty"`
	Version          int       `json:"version"`
}

type Comment struct {
	ID          uuid.UUID `json:"id"`           // Unique identifier for the comment
	IdeaID      uuid.UUID `json:"idea_id"`      // ID of the idea the comment is related to
	CommentedBy uuid.UUID `json:"commented_by"` // User ID of the person who made the comment
	Content     string    `json:"content"`      // Content of the comment
	CreatedAt   time.Time `json:"created_at"`   // Timestamp of when the comment was created
}

func ValidateIdea(v *validator.Validator, idea *Idea) {
	v.Check(idea.Title != "", "title", "must be provided")
	v.Check(len(idea.Title) <= 100, "title", "must not be more than 100 bytes long")

	v.Check(idea.Description != "", "description", "must be provided")
	v.Check(len(idea.Description) <= 1000, "description", "must not be more than 1000 bytes long")

	// v.Check(idea.Pdf != "", "pdf", "must be provided")
	// v.Check(len(idea.Pdf) > 0, "pdf", "must not be empty")

	// isPDF := strings.EqualFold(filepath.Ext(idea.Pdf), ".pdf")

	// v.Check(isPDF, "pdf", "must be a PDF file")

	// fileInfo, err := os.Stat(idea.Pdf)
	// if err != nil {
	// 	v.AddError("pdf", "unable to get file into")
	// } else {
	// 	const maxPDFSize = 5 * 1024 * 1024 // 5MB
	// 	v.Check(fileInfo.Size() <= maxPDFSize, "pdf", "file size must be less than 5MB")
	// }

	v.Check(idea.Category != "", "category", "must be provided")
	v.Check(len(idea.Category) <= 50, "category", "must not be more than 50 bytes long")

	v.Check(idea.Tags != nil, "tags", "must be provided")
	v.Check(len(idea.Tags) >= 1, "tags", "must contain at least one tag")
	v.Check(validator.Unique(idea.Tags), "tags", "must not contain duplicate values")

	// v.Check(idea.SubmittedBy != 0, "submitted_by", "must be provided")
	// v.Check(idea.SubmittedBy > 0, "submitted_by", "must be a positive integer")
	if !ValidateUUID(idea.UserID.String()) {
		v.AddError("user_id", "must be a valid UUID")
	}
	// if !ValidateUUID(idea.IdeaSourceID.String()) {
	// 		v.AddError("idea_source_id", "must be a valid UUID")
	// 	}

	if idea.GitHubLink != "" {
		v.Check(IsValidURL(idea.GitHubLink), "github_link", "must be a valid URL")
	}
	if idea.WebsiteLink != "" {
		v.Check(IsValidURL(idea.WebsiteLink), "website_link", "must be a valid URL")
	}

	v.Check(idea.LearningOutcome != "", "learning_outcome", "must be provided")

	v.Check(len(idea.LearningOutcome) <= 1000, "learning_outcome", "must not be more than 1000 bytes long")

	v.Check(idea.RecommendedLevel != "", "recommended_level", "must be provided")

	v.Check(len(idea.RecommendedLevel) <= 50, "recommended_level", "must not be more than 50 bytes long")
}

func IsValidURL(str string) bool {
	return len(str) > 0 && (strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://"))
}

func ValidateUUID(uuidStr string) bool {
	_, err := uuid.Parse(uuidStr)
	return err == nil
}

type IdeaModel struct {
	DB *sql.DB
}

func (i IdeaModel) Insert(idea *Idea) error {

	Status := "pending"

	query := `INSERT INTO ideas (title, description, user_id, idea_source_id, category, tags,
	learning_outcome, recommended_level, github_link, website_link, status)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) 
	RETURNING id, created_at, updated_at, version`

	args := []any{
		idea.Title,
		idea.Description,
		idea.UserID,
		idea.IdeaSourceID,
		idea.Category,
		pq.Array(idea.Tags),
		idea.LearningOutcome,
		idea.RecommendedLevel,
		idea.GitHubLink,
		idea.WebsiteLink,
		Status,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return i.DB.QueryRowContext(ctx, query, args...).Scan(&idea.ID, &idea.CreatedAt, &idea.UpdatedAt, &idea.Version)

}

func (i IdeaModel) Update(idea *Idea) error {

	if idea.FeedbackStr != "" {
        idea.Feedback = sql.NullString{
            String: idea.FeedbackStr,
            Valid: true,
        }
    }
	query := `UPDATE ideas 
              SET title = $1, description = $2, user_id = $3, idea_source_id = $4, 
                  category = $5, tags = $6, learning_outcome = $7, recommended_level = $8,
                  github_link = $9, website_link = $10,
				  feedback = $11, status = $12, version = version + 1 
              WHERE id = $13 AND version = $14
              RETURNING version`

	args := []any{
		idea.Title,
		idea.Description,
		idea.UserID,
		idea.IdeaSourceID,
		idea.Category,
		pq.Array(idea.Tags),
		idea.LearningOutcome,
		idea.RecommendedLevel,
		idea.GitHubLink,
		idea.WebsiteLink,
		idea.Feedback,
		idea.Status,
		idea.ID,
		idea.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := i.DB.QueryRowContext(ctx, query, args...).Scan(&idea.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (i IdeaModel) Get(id uuid.UUID) (*Idea, error) {
	query := `SELECT id, created_at, updated_at, title, description, user_id, idea_source_id, 
                    category, tags, status, learning_outcome, recommended_level, github_link, feedback,
                    website_link, version 
             FROM ideas 
             WHERE id = $1`

	var idea Idea

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := i.DB.QueryRowContext(ctx, query, id).Scan(
		&idea.ID,
		&idea.CreatedAt,
		&idea.UpdatedAt,
		&idea.Title,
		&idea.Description,
		&idea.UserID,
		&idea.IdeaSourceID,
		&idea.Category,
		pq.Array(&idea.Tags),
		&idea.Status,
		&idea.LearningOutcome,
		&idea.RecommendedLevel,
		&idea.GitHubLink,
		&idea.Feedback,
		&idea.WebsiteLink,
		&idea.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	// Check if the feedback is null and set it to an empty string if it is

	if idea.Feedback.Valid {
		idea.FeedbackStr = idea.Feedback.String
	} else {
		idea.FeedbackStr = ""
	}

	return &idea, nil

}

func (i IdeaModel) Delete(id uuid.UUID) error {
	query := `DELETE FROM ideas WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := i.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (i IdeaModel) GetAllIdeas(title string, tags []string, filters Filters) ([]*Idea, Metadata, error) {
	query := fmt.Sprintf(`SELECT count(*) OVER(), id, created_at, updated_at, title, description, 
                                user_id, idea_source_id, category, tags, status, learning_outcome, 
                                recommended_level, github_link, website_link, feedback, version
                          FROM ideas 
                          WHERE (to_tsvector('english', title) @@ plainto_tsquery('english', $1) OR $1 = '') 
                          AND (tags @> $2 OR $2 = '{}') 
                          ORDER BY %s %s, id ASC
                          LIMIT $3 OFFSET $4`,
		filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{title, pq.Array(tags), filters.limit(), filters.offset()}

	rows, err := i.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	ideas := []*Idea{}

	for rows.Next() {
		var idea Idea

		err := rows.Scan(
			&totalRecords,
			&idea.ID,
			&idea.CreatedAt,
			&idea.UpdatedAt,
			&idea.Title,
			&idea.Description,
			&idea.UserID,
			&idea.IdeaSourceID,
			&idea.Category,
			pq.Array(&idea.Tags),
			&idea.Status,
			&idea.LearningOutcome,
			&idea.RecommendedLevel,
			&idea.GitHubLink,
			&idea.WebsiteLink,
			&idea.Feedback,
			&idea.Version,
		)

		if err != nil {
			return nil, Metadata{}, err
		}

		ideas = append(ideas, &idea)

	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return ideas, metadata, nil
}

func (i IdeaModel) GetAllByUserID(userID uuid.UUID, limit, offset int) ([]*Idea, int, error) {
	// Query to get total count
	countQuery := `
        SELECT COUNT(*) 
        FROM ideas 
        WHERE user_id = $1`

	// Main query with pagination
	query := `
        SELECT id, created_at, updated_at, title, description, user_id, idea_source_id, 
               category, tags, status, learning_outcome, recommended_level, github_link,
               website_link, feedback, version
        FROM ideas
        WHERE user_id = $1
        ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var totalCount int
	err := i.DB.QueryRowContext(ctx, countQuery, userID).Scan(&totalCount)

	if err != nil {
		return nil, 0, err
	}

	rows, err := i.DB.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	var ideas []*Idea

	for rows.Next() {
		var idea Idea

		err := rows.Scan(
			&idea.ID,
			&idea.CreatedAt,
			&idea.UpdatedAt,
			&idea.Title,
			&idea.Description,
			&idea.UserID,
			&idea.IdeaSourceID,
			&idea.Category,
			pq.Array(&idea.Tags),
			&idea.Status,
			&idea.LearningOutcome,
			&idea.RecommendedLevel,
			&idea.GitHubLink,
			&idea.WebsiteLink,
			&idea.Feedback,
			&idea.Version,
		)

		if err != nil {
			return nil, 0, err
		}

		ideas = append(ideas, &idea)

	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return ideas, totalCount, nil

}

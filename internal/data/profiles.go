package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Profile struct {
	UserID    uuid.UUID `json:"user_id"`
	Firstname string    `json:"firstname,omitempty"`
	Lastname  string    `json:"lastname,omitempty"`
	Avatar    string    `json:"avatar,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Title     string    `json:"title,omitempty"`
	Bio       string    `json:"bio,omitempty"`
	Faculty   string    `json:"faculty,omitempty"`
	Program   string    `json:"program,omitempty"`
	Degree    string    `json:"degree,omitempty"`
	Year      string    `json:"year,omitempty"`
	Uni       string    `json:"uni,omitempty"`
	Mobile    string    `json:"mobile,omitempty"`
	LinkedIn  string    `json:"linkedin,omitempty"`
	GitHub    string    `json:"github,omitempty"`
	FB        string    `json:"fb,omitempty"`
	Skills    []string  `json:"skills"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserProfile struct {
	ID                  uuid.UUID `json:"id"`
	Username            string    `json:"username"`
	Email               string    `json:"email"`
	UserType            string    `json:"user_type"`
	Firstname           string    `json:"firstname,omitempty"`
	Lastname            string    `json:"lastname,omitempty"`
	Year                string    `json:"year,omitempty"`
	Avatar              string    `json:"avatar,omitempty"`
	AvatarURL           string    `json:"avatar_url,omitempty"`
	Title               string    `json:"title,omitempty"`
	Bio                 string    `json:"bio,omitempty"`
	Faculty             string    `json:"faculty,omitempty"`
	Program             string    `json:"program,omitempty"`
	Degree              string    `json:"degree,omitempty"`
	Uni                 string    `json:"uni,omitempty"`
	Mobile              string    `json:"mobile,omitempty"`
	LinkedIn            string    `json:"linkedin,omitempty"`
	GitHub              string    `json:"github,omitempty"`
	FB                  string    `json:"fb,omitempty"`
	Skills              []string  `json:"skills"`
	HasCompletedProfile bool      `json:"has_completed_profile"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type UserProfileWithIdeas struct {
	UserProfile *UserProfile `json:"profile"`
	Ideas       []*Idea      `json:"ideas"`
}

type ProfileModel struct {
	DB *sql.DB
}

func (m ProfileModel) GetAllProfilesWithIdeas(limit int, offset int) ([]*UserProfileWithIdeas, error) {
	query := `
        SELECT u.id, u.user_name, u.email, u.user_type, u.created_at,
        u.has_profile_created,
               p.firstname, p.lastname, p.avatar, p.title, p.bio, 
               p.faculty, p.program, p.degree, p.year, p.uni, p.mobile, 
               p.linkedin, p.github, p.fb, p.updated_at
        FROM users u
        LEFT JOIN user_profiles p ON u.id = p.user_id
        WHERE u.has_profile_created = true AND u.user_type != 'admin'
        ORDER BY u.created_at DESC
        LIMIT $1 OFFSET $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profilesWithIdeas []*UserProfileWithIdeas

	for rows.Next() {
		var profile UserProfile
		var hasProfileCreated bool
		var firstname, lastname, avatar, title, bio sql.NullString
		var faculty, program, degree, year, uni, mobile sql.NullString
		var linkedin, github, fb sql.NullString
		var updatedAt sql.NullTime

		err := rows.Scan(
			&profile.ID,
			&profile.Username,
			&profile.Email,
			&profile.UserType,
			&profile.CreatedAt,
			&hasProfileCreated,
			&firstname,
			&lastname,
			&avatar,
			&title,
			&bio,
			&faculty,
			&program,
			&degree,
			&year,
			&uni,
			&mobile,
			&linkedin,
			&github,
			&fb,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Set nullable fields
		if firstname.Valid {
			profile.Firstname = firstname.String
		}
		if lastname.Valid {
			profile.Lastname = lastname.String
		}
		if avatar.Valid {
			profile.Avatar = avatar.String
		}
		if title.Valid {
			profile.Title = title.String
		}
		if bio.Valid {
			profile.Bio = bio.String
		}
		if faculty.Valid {
			profile.Faculty = faculty.String
		}
		if program.Valid {
			profile.Program = program.String
		}
		if degree.Valid {
			profile.Degree = degree.String
		}
		if year.Valid {
			profile.Year = year.String
		}
		if uni.Valid {
			profile.Uni = uni.String
		}
		if mobile.Valid {
			profile.Mobile = mobile.String
		}
		if linkedin.Valid {
			profile.LinkedIn = linkedin.String
		}
		if github.Valid {
			profile.GitHub = github.String
		}
		if fb.Valid {
			profile.FB = fb.String
		}
		if updatedAt.Valid {
			profile.UpdatedAt = updatedAt.Time
		} else {
			profile.UpdatedAt = profile.CreatedAt
		}

		profile.HasCompletedProfile = hasProfileCreated

		if avatar.Valid && avatar.String != "" && avatar.String != "no key" {
			profile.Avatar = avatar.String
			profile.AvatarURL = "/v1/avatars/" + avatar.String
		}
		// Get skills for the profile
		skillsQuery := `
            SELECT skill
            FROM user_skills
            WHERE user_id = $1
        `
		skillRows, err := m.DB.QueryContext(ctx, skillsQuery, profile.ID)
		if err != nil {
			return nil, err
		}

		var skills []string
		for skillRows.Next() {
			var skill string
			err := skillRows.Scan(&skill)
			if err != nil {
				skillRows.Close()
				return nil, err
			}
			skills = append(skills, skill)
		}
		skillRows.Close()
		if err = skillRows.Err(); err != nil {
			return nil, err
		}

		profile.Skills = skills

		// Get ideas submitted by the user
		ideasQuery := `
    SELECT id, created_at, updated_at, title, description, user_id, idea_source_id,
           category, tags, status, learning_outcome, recommended_level, github_link,
           website_link, version
    FROM ideas
    WHERE user_id = $1
    ORDER BY created_at DESC
`
		ideaRows, err := m.DB.QueryContext(ctx, ideasQuery, profile.ID)
		if err != nil {
			return nil, err
		}

		// Initialize an empty slice for ideas - this ensures that users with no ideas
		// still have an empty array in the JSON response instead of null
		ideas := []*Idea{}

		for ideaRows.Next() {
			var idea Idea
			var tags []string

			err := ideaRows.Scan(
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
				&idea.Version,
			)
			if err != nil {
				ideaRows.Close()
				return nil, err
			}

			idea.Tags = tags
			idea.UserID = profile.ID
			ideas = append(ideas, &idea)
		}
		ideaRows.Close()
		if err = ideaRows.Err(); err != nil {
			return nil, err
		}

		// Always add the user to the result, even if they have no ideas
		profilesWithIdeas = append(profilesWithIdeas, &UserProfileWithIdeas{
			UserProfile: &profile,
			Ideas:       ideas,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return profilesWithIdeas, nil
}
func (m ProfileModel) GetByUserID(userID uuid.UUID) (*Profile, error) {
	query := `
		SELECT user_id, firstname, lastname, avatar, title, bio, faculty, 
		       program, degree, year, uni, mobile, linkedin, github, fb, 
		       created_at, updated_at
		FROM user_profiles
		WHERE user_id = $1`

	var profile Profile

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, userID).Scan(
		&profile.UserID,
		&profile.Firstname,
		&profile.Lastname,
		&profile.Avatar,
		&profile.Title,
		&profile.Bio,
		&profile.Faculty,
		&profile.Program,
		&profile.Degree,
		&profile.Year,
		&profile.Uni,
		&profile.Mobile,
		&profile.LinkedIn,
		&profile.GitHub,
		&profile.FB,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	// Get skills for the profile
	query = `
		SELECT skill
		FROM user_skills
		WHERE user_id = $1
	`

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []string
	for rows.Next() {
		var skill string
		err := rows.Scan(&skill)

		if err != nil {
			return nil, err
		}

		skills = append(skills, skill)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	profile.Skills = skills

	return &profile, nil

}

func (m ProfileModel) GetUserProfile(userID uuid.UUID) (*UserProfile, error) {
	query := `
		SELECT u.id, u.user_name, u.email, u.user_type, u.created_at, u.has_profile_created,
		       p.firstname, p.lastname, p.avatar, p.title, p.bio, 
		       p.faculty, p.program, p.degree, p.year, p.uni, p.mobile, 
		       p.linkedin, p.github, p.fb, p.updated_at
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
		WHERE u.id = $1`

	var profile UserProfile
	var hasProfileCreated bool
	var firstname, lastname, avatar, title, bio sql.NullString
	var faculty, program, degree, year, uni, mobile sql.NullString
	var linkedin, github, fb sql.NullString
	var updatedAt sql.NullTime

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, userID).Scan(
		&profile.ID,
		&profile.Username,
		&profile.Email,
		&profile.UserType,
		&profile.CreatedAt,
		&hasProfileCreated,
		&firstname,
		&lastname,
		&avatar,
		&title,
		&bio,
		&faculty,
		&program,
		&degree,
		&year,
		&uni,
		&mobile,
		&linkedin,
		&github,
		&fb,
		&updatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	profile.HasCompletedProfile = hasProfileCreated
	// Set nullable fields
	if firstname.Valid {
		profile.Firstname = firstname.String
	}
	if lastname.Valid {
		profile.Lastname = lastname.String
	}
	if avatar.Valid {
		profile.Avatar = avatar.String
	}
	if title.Valid {
		profile.Title = title.String
	}
	if bio.Valid {
		profile.Bio = bio.String
	}
	if faculty.Valid {
		profile.Faculty = faculty.String
	}
	if program.Valid {
		profile.Program = program.String
	}
	if degree.Valid {
		profile.Degree = degree.String
	}
	if year.Valid {
		profile.Year = year.String
	}
	if uni.Valid {
		profile.Uni = uni.String
	}
	if mobile.Valid {
		profile.Mobile = mobile.String
	}
	if linkedin.Valid {
		profile.LinkedIn = linkedin.String
	}
	if github.Valid {
		profile.GitHub = github.String
	}
	if fb.Valid {
		profile.FB = fb.String
	}
	if updatedAt.Valid {
		profile.UpdatedAt = updatedAt.Time
	} else {
		profile.UpdatedAt = profile.CreatedAt
	}

	if avatar.Valid && avatar.String != "" && avatar.String != "no key" {
		profile.Avatar = avatar.String
		profile.AvatarURL = "/v1/avatars/" + avatar.String
	}
	// Get skills for the profile
	query = `
		SELECT skill
		FROM user_skills
		WHERE user_id = $1
	`

	rows, err := m.DB.QueryContext(ctx, query, profile.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []string
	for rows.Next() {
		var skill string
		err := rows.Scan(&skill)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	profile.Skills = skills

	
	return &profile, nil
}

func (m ProfileModel) GetFullProfile(userID uuid.UUID) (*UserProfile, error) {
	query := `
		SELECT u.id, u.user_name, u.email, u.user_type, u.created_at, u.has_profile_created,
		       p.firstname, p.lastname, p.avatar, p.title, p.bio, 
		       p.faculty, p.program, p.degree, p.year, p.uni, p.mobile, 
		       p.linkedin, p.github, p.fb, p.updated_at
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
		WHERE u.has_profile_created = true AND u.id = $1`

	var profile UserProfile
	var hasProfileCreated bool
	var firstname, lastname, avatar, title, bio sql.NullString
	var faculty, program, degree, year, uni, mobile sql.NullString
	var linkedin, github, fb sql.NullString
	var updatedAt sql.NullTime

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, userID).Scan(
		&profile.ID,
		&profile.Username,
		&profile.Email,
		&profile.UserType,
		&profile.CreatedAt,
		&hasProfileCreated,
		&firstname,
		&lastname,
		&avatar,
		&title,
		&bio,
		&faculty,
		&program,
		&degree,
		&year,
		&uni,
		&mobile,
		&linkedin,
		&github,
		&fb,
		&updatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	profile.HasCompletedProfile = hasProfileCreated
	// Set nullable fields
	if firstname.Valid {
		profile.Firstname = firstname.String
	}
	if lastname.Valid {
		profile.Lastname = lastname.String
	}
	if avatar.Valid {
		profile.Avatar = avatar.String
	}
	if title.Valid {
		profile.Title = title.String
	}
	if bio.Valid {
		profile.Bio = bio.String
	}
	if faculty.Valid {
		profile.Faculty = faculty.String
	}
	if program.Valid {
		profile.Program = program.String
	}
	if degree.Valid {
		profile.Degree = degree.String
	}
	if year.Valid {
		profile.Year = year.String
	}
	if uni.Valid {
		profile.Uni = uni.String
	}
	if mobile.Valid {
		profile.Mobile = mobile.String
	}
	if linkedin.Valid {
		profile.LinkedIn = linkedin.String
	}
	if github.Valid {
		profile.GitHub = github.String
	}
	if fb.Valid {
		profile.FB = fb.String
	}
	if updatedAt.Valid {
		profile.UpdatedAt = updatedAt.Time
	} else {
		profile.UpdatedAt = profile.CreatedAt
	}

	if avatar.Valid && avatar.String != "" && avatar.String != "no key" {
		profile.Avatar = avatar.String
		profile.AvatarURL = "/v1/avatars/" + avatar.String
	}
	// Get skills for the profile
	query = `
		SELECT skill
		FROM user_skills
		WHERE user_id = $1
	`

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []string
	for rows.Next() {
		var skill string
		err := rows.Scan(&skill)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	profile.Skills = skills

	// Determine if profile is complete
	profile.HasCompletedProfile = profile.Firstname != "" &&
		profile.Lastname != "" &&
		profile.Title != "" &&
		profile.Bio != ""

	return &profile, nil
}

func (m ProfileModel) GetByUsername(username string) (*UserProfile, error) {
	query := `
		SELECT u.id, u.user_name, u.email, u.user_type, u.created_at,
		       p.firstname, p.lastname, p.avatar, p.title, p.bio, 
		       p.faculty, p.program, p.degree, p.year, p.uni, p.mobile, 
		       p.linkedin, p.github, p.fb, p.updated_at
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
		WHERE u.user_name = $1`

	var profile UserProfile
	var firstname, lastname, avatar, title, bio sql.NullString
	var faculty, program, degree, year, uni, mobile sql.NullString
	var linkedin, github, fb sql.NullString
	var updatedAt sql.NullTime

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, username).Scan(
		&profile.ID,
		&profile.Username,
		&profile.Email,
		&profile.UserType,
		&profile.CreatedAt,
		&firstname,
		&lastname,
		&avatar,
		&title,
		&bio,
		&faculty,
		&program,
		&degree,
		&year,
		&uni,
		&mobile,
		&linkedin,
		&github,
		&fb,
		&updatedAt,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	// Set nullable fields (same as in GetFullProfile)
	// [same code as above for setting nullable fields]

	// Get skills
	// [same code as above for getting skills]

	if firstname.Valid {
		profile.Firstname = firstname.String
	}
	if lastname.Valid {
		profile.Lastname = lastname.String
	}
	if avatar.Valid {
		profile.Avatar = avatar.String
	}
	if title.Valid {
		profile.Title = title.String
	}
	if bio.Valid {
		profile.Bio = bio.String
	}
	if faculty.Valid {
		profile.Faculty = faculty.String
	}
	if program.Valid {
		profile.Program = program.String
	}
	if degree.Valid {
		profile.Degree = degree.String
	}
	if year.Valid {
		profile.Year = year.String
	}
	if uni.Valid {
		profile.Uni = uni.String
	}
	if mobile.Valid {
		profile.Mobile = mobile.String
	}
	if linkedin.Valid {
		profile.LinkedIn = linkedin.String
	}
	if github.Valid {
		profile.GitHub = github.String
	}
	if fb.Valid {
		profile.FB = fb.String
	}
	if updatedAt.Valid {
		profile.UpdatedAt = updatedAt.Time
	} else {
		profile.UpdatedAt = profile.CreatedAt
	}

	if avatar.Valid && avatar.String != "" && avatar.String != "no key" {
		profile.Avatar = avatar.String
		profile.AvatarURL = "/v1/avatars/" + avatar.String
	}

	// Get skills for the profile
	query = `
		SELECT skill
		FROM user_skills
		WHERE user_id = $1
	`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []string
	for rows.Next() {
		var skill string
		err := rows.Scan(&skill)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	profile.Skills = skills

	// Determine if profile is complete
	profile.HasCompletedProfile = profile.Firstname != "" &&
		profile.Lastname != "" &&
		profile.Title != "" &&
		profile.Bio != ""

	return &profile, nil
}

func (m ProfileModel) Insert(profile *Profile) error {
	query := `
		INSERT INTO user_profiles 
		(user_id, firstname, lastname, avatar, title, bio, faculty, program, degree, year, uni, mobile, linkedin, github, fb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING created_at, updated_at`

	args := []any{
		profile.UserID,
		profile.Firstname,
		profile.Lastname,
		profile.Avatar,
		profile.Title,
		profile.Bio,
		profile.Faculty,
		profile.Program,
		profile.Degree,
		profile.Year,
		profile.Uni,
		profile.Mobile,
		profile.LinkedIn,
		profile.GitHub,
		profile.FB,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return err
	}

	// Insert skills
	if len(profile.Skills) > 0 {
		err = m.updateSkills(profile.UserID, profile.Skills)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m ProfileModel) Update(profile *Profile) error {
	query := `
		UPDATE user_profiles
		SET firstname = $1, lastname = $2, avatar = $3, title = $4, bio = $5,
		    faculty = $6, program = $7, degree = $8, year=$9, uni = $10, mobile = $11,
		    linkedin = $12, github = $13, fb = $14,  updated_at = NOW()
		WHERE user_id = $15
		RETURNING updated_at`

	args := []any{
		profile.Firstname,
		profile.Lastname,
		profile.Avatar,
		profile.Title,
		profile.Bio,
		profile.Faculty,
		profile.Program,
		profile.Degree,
		profile.Year,
		profile.Uni,
		profile.Mobile,
		profile.LinkedIn,
		profile.GitHub,
		profile.FB,
		profile.UserID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&profile.UpdatedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// Create profile if it doesn't exist
			return m.Insert(profile)
		default:
			return err
		}
	}

	// Update skills
	if profile.Skills != nil {
		err = m.updateSkills(profile.UserID, profile.Skills)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m ProfileModel) updateSkills(userID uuid.UUID, skills []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Begin transaction
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing skills
	_, err = tx.ExecContext(ctx, "DELETE FROM user_skills WHERE user_id = $1", userID)
	if err != nil {
		return err
	}

	// Insert new skills
	for _, skill := range skills {
		_, err = tx.ExecContext(ctx,
			"INSERT INTO user_skills (user_id, skill) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			userID, skill)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m ProfileModel) UpdateSkills(userID uuid.UUID, skills []string) error {
	return m.updateSkills(userID, skills)
}

func (m ProfileModel) Search(query string, limit int, offset int) ([]*UserProfile, error) {
	sqlQuery := `
		SELECT u.id, u.user_name, u.email, u.user_type, u.created_at,
		       p.firstname, p.lastname, p.avatar, p.title, p.bio, 
		       p.faculty, p.program, p.degree, p.year, p.uni, p.mobile, 
		       p.linkedin, p.github, p.fb, p.updated_at
		FROM users u
		LEFT JOIN user_profiles p ON u.id = p.user_id
		WHERE u.user_name ILIKE $1
		   OR p.firstname ILIKE $1
		   OR p.lastname ILIKE $1
		   OR p.title ILIKE $1
		   OR p.bio ILIKE $1
		   OR p.faculty ILIKE $1
		   OR p.program ILIKE $1
		ORDER BY 
		    CASE WHEN u.user_name ILIKE $1 THEN 1
		         WHEN p.firstname ILIKE $1 OR p.lastname ILIKE $1 THEN 2
		         ELSE 3
		    END,
		    u.created_at DESC
		LIMIT $2 OFFSET $3`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	searchTerm := "%" + query + "%"
	rows, err := m.DB.QueryContext(ctx, sqlQuery, searchTerm, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := []*UserProfile{}

	for rows.Next() {
		var profile UserProfile
		var firstname, lastname, avatar, title, bio sql.NullString
		var faculty, program, degree, year, uni, mobile sql.NullString
		var linkedin, github, fb sql.NullString
		var updatedAt sql.NullTime

		err := rows.Scan(
			&profile.ID,
			&profile.Username,
			&profile.Email,
			&profile.UserType,
			&profile.CreatedAt,
			&firstname,
			&lastname,
			&avatar,
			&title,
			&bio,
			&faculty,
			&program,
			&degree,
			&year,
			&uni,
			&mobile,
			&linkedin,
			&github,
			&fb,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Set nullable fields
		if firstname.Valid {
			profile.Firstname = firstname.String
		}
		if lastname.Valid {
			profile.Lastname = lastname.String
		}
		if avatar.Valid {
			profile.Avatar = avatar.String
		}
		if title.Valid {
			profile.Title = title.String
		}
		if bio.Valid {
			profile.Bio = bio.String
		}
		if faculty.Valid {
			profile.Faculty = faculty.String
		}
		if program.Valid {
			profile.Program = program.String
		}
		if degree.Valid {
			profile.Degree = degree.String
		}
		if year.Valid {
			profile.Year = year.String
		}
		if uni.Valid {
			profile.Uni = uni.String
		}
		if mobile.Valid {
			profile.Mobile = mobile.String
		}
		if linkedin.Valid {
			profile.LinkedIn = linkedin.String
		}
		if github.Valid {
			profile.GitHub = github.String
		}
		if fb.Valid {
			profile.FB = fb.String
		}
		if updatedAt.Valid {
			profile.UpdatedAt = updatedAt.Time
		} else {
			profile.UpdatedAt = profile.CreatedAt
		}

		// Determine if profile is complete
		profile.HasCompletedProfile = profile.Firstname != "" &&
			profile.Lastname != "" &&
			profile.Title != "" &&
			profile.Bio != ""

		// Get skills in a separate query
		skillsQuery := `
			SELECT skill
			FROM user_skills
			WHERE user_id = $1
		`
		skillRows, err := m.DB.QueryContext(ctx, skillsQuery, profile.ID)
		if err != nil {
			return nil, err
		}

		var skills []string
		for skillRows.Next() {
			var skill string
			err := skillRows.Scan(&skill)
			if err != nil {
				skillRows.Close()
				return nil, err
			}
			skills = append(skills, skill)
		}
		skillRows.Close()
		if err = skillRows.Err(); err != nil {
			return nil, err
		}

		profile.Skills = skills
		profiles = append(profiles, &profile)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return profiles, nil
}

func (m ProfileModel) Delete(userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Begin transaction
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete skills first (cascading would handle this, but we'll be explicit)
	_, err = tx.ExecContext(ctx, "DELETE FROM user_skills WHERE user_id = $1", userID)
	if err != nil {
		return err
	}

	// Delete the profile
	result, err := tx.ExecContext(ctx, "DELETE FROM user_profiles WHERE user_id = $1", userID)
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

	return tx.Commit()
}

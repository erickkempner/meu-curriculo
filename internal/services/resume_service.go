package services

import (
	"context"
	"fmt"

	"github.com/erick/curriculo/internal/db"
	"github.com/erick/curriculo/internal/models"
	"github.com/erick/curriculo/internal/repositories"
	"github.com/erick/curriculo/internal/validators"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateResumeInput holds the data required to create a new resume.
type CreateResumeInput struct {
	Title         string
	TemplateName  string
	PersonalName  string
	PersonalTitle string
	Email         string
	Phone         string
	Location      string
	Summary       string
	PhotoURL      string
	Experience    []ExperienceInput
	Education     []EducationInput
	Skills        []string
}

// UpdateResumeInput is an alias for CreateResumeInput since updates use the same fields.
type UpdateResumeInput = CreateResumeInput

// ExperienceInput holds the data for a single experience entry.
type ExperienceInput struct {
	Company     string
	Role        string
	Period      string
	Description string
}

// EducationInput holds the data for a single education entry.
type EducationInput struct {
	Institution string
	Degree      string
	Period      string
}

// ResumeService defines the business logic operations for resumes.
type ResumeService interface {
	List(ctx context.Context, userID uuid.UUID) ([]models.Resume, error)
	GetByID(ctx context.Context, userID, resumeID uuid.UUID) (*models.ResumeDetail, error)
	Create(ctx context.Context, userID uuid.UUID, input CreateResumeInput) (*models.Resume, error)
	Update(ctx context.Context, userID, resumeID uuid.UUID, input UpdateResumeInput) error
	Delete(ctx context.Context, userID, resumeID uuid.UUID) error
	Duplicate(ctx context.Context, userID, resumeID uuid.UUID) (*models.Resume, error)
	GenerateShareToken(ctx context.Context, userID, resumeID uuid.UUID) (string, error)
	RevokeShareToken(ctx context.Context, userID, resumeID uuid.UUID) error
	RegenerateShareToken(ctx context.Context, userID, resumeID uuid.UUID) (string, error)
	GetByShareToken(ctx context.Context, token string) (*models.ResumeDetail, error)
	UpdatePhotoURL(ctx context.Context, userID, resumeID uuid.UUID, photoURL string) error
	UpdateThumbnailURL(ctx context.Context, userID, resumeID uuid.UUID, thumbnailURL string) error
}

// resumeService implements ResumeService using a ResumeRepository.
type resumeService struct {
	resumeRepo repositories.ResumeRepository
}

// NewResumeService creates a new ResumeService backed by the given ResumeRepository.
func NewResumeService(resumeRepo repositories.ResumeRepository) ResumeService {
	return &resumeService{resumeRepo: resumeRepo}
}

// List returns all resumes belonging to the given user, ordered by updated_at descending.
func (s *resumeService) List(ctx context.Context, userID uuid.UUID) ([]models.Resume, error) {
	dbResumes, err := s.resumeRepo.FindAllByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	resumes := make([]models.Resume, len(dbResumes))
	for i, r := range dbResumes {
		resumes[i] = dbResumeToModel(r)
	}
	return resumes, nil
}

// GetByID finds a resume by ID, verifies ownership, and loads all children.
func (s *resumeService) GetByID(ctx context.Context, userID, resumeID uuid.UUID) (*models.ResumeDetail, error) {
	dbResume, err := s.resumeRepo.FindByID(ctx, resumeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}

	if err := s.verifyOwnership(dbResume, userID); err != nil {
		return nil, err
	}

	return s.loadResumeDetail(ctx, dbResume)
}

// Create validates input, sanitizes text fields, and persists the resume with children.
func (s *resumeService) Create(ctx context.Context, userID uuid.UUID, input CreateResumeInput) (*models.Resume, error) {
	if err := validateResumeInput(input); err != nil {
		return nil, err
	}

	// Sanitize all text fields
	sanitized := sanitizeResumeInput(input)

	// Create resume record
	dbResume, err := s.resumeRepo.Create(ctx, db.CreateResumeParams{
		UserID:        uuidToPgtype(userID),
		Title:         sanitized.Title,
		TemplateName:  sanitized.TemplateName,
		PersonalName:  sanitized.PersonalName,
		PersonalTitle: sanitized.PersonalTitle,
		Email:         sanitized.Email,
		Phone:         sanitized.Phone,
		Location:      sanitized.Location,
		Summary:       sanitized.Summary,
		PhotoUrl:      sanitized.PhotoURL,
	})
	if err != nil {
		return nil, err
	}

	resumeID := uuid.UUID(dbResume.ID.Bytes)

	// Create experience entries
	for i, exp := range sanitized.Experience {
		_, err := s.resumeRepo.CreateExperience(ctx, db.CreateExperienceParams{
			ResumeID:     dbResume.ID,
			Company:      exp.Company,
			Role:         exp.Role,
			Period:       exp.Period,
			Description:  exp.Description,
			DisplayOrder: int32(i),
		})
		if err != nil {
			return nil, fmt.Errorf("creating experience %d: %w", i, err)
		}
	}

	// Create education entries
	for i, edu := range sanitized.Education {
		_, err := s.resumeRepo.CreateEducation(ctx, db.CreateEducationParams{
			ResumeID:     dbResume.ID,
			Institution:  edu.Institution,
			Degree:       edu.Degree,
			Period:       edu.Period,
			DisplayOrder: int32(i),
		})
		if err != nil {
			return nil, fmt.Errorf("creating education %d: %w", i, err)
		}
	}

	// Create skill entries
	for i, skill := range sanitized.Skills {
		_, err := s.resumeRepo.CreateSkill(ctx, db.CreateSkillParams{
			ResumeID:     dbResume.ID,
			Name:         skill,
			DisplayOrder: int32(i),
		})
		if err != nil {
			return nil, fmt.Errorf("creating skill %d: %w", i, err)
		}
	}

	result := dbResumeToModel(dbResume)
	result.ID = resumeID
	return &result, nil
}

// Update verifies ownership, validates input, deletes old children, and inserts new ones.
func (s *resumeService) Update(ctx context.Context, userID, resumeID uuid.UUID, input UpdateResumeInput) error {
	dbResume, err := s.resumeRepo.FindByID(ctx, resumeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.ErrNotFound
		}
		return err
	}

	if err := s.verifyOwnership(dbResume, userID); err != nil {
		return err
	}

	if err := validateResumeInput(input); err != nil {
		return err
	}

	sanitized := sanitizeResumeInput(input)

	// Update resume fields
	err = s.resumeRepo.Update(ctx, db.UpdateResumeParams{
		ID:            uuidToPgtype(resumeID),
		Title:         sanitized.Title,
		TemplateName:  sanitized.TemplateName,
		PersonalName:  sanitized.PersonalName,
		PersonalTitle: sanitized.PersonalTitle,
		Email:         sanitized.Email,
		Phone:         sanitized.Phone,
		Location:      sanitized.Location,
		Summary:       sanitized.Summary,
		PhotoUrl:      sanitized.PhotoURL,
	})
	if err != nil {
		return err
	}

	// Delete old children
	if err := s.resumeRepo.DeleteExperiencesByResumeID(ctx, resumeID); err != nil {
		return err
	}
	if err := s.resumeRepo.DeleteEducationsByResumeID(ctx, resumeID); err != nil {
		return err
	}
	if err := s.resumeRepo.DeleteSkillsByResumeID(ctx, resumeID); err != nil {
		return err
	}

	pgResumeID := uuidToPgtype(resumeID)

	// Insert new experience entries
	for i, exp := range sanitized.Experience {
		_, err := s.resumeRepo.CreateExperience(ctx, db.CreateExperienceParams{
			ResumeID:     pgResumeID,
			Company:      exp.Company,
			Role:         exp.Role,
			Period:       exp.Period,
			Description:  exp.Description,
			DisplayOrder: int32(i),
		})
		if err != nil {
			return fmt.Errorf("creating experience %d: %w", i, err)
		}
	}

	// Insert new education entries
	for i, edu := range sanitized.Education {
		_, err := s.resumeRepo.CreateEducation(ctx, db.CreateEducationParams{
			ResumeID:     pgResumeID,
			Institution:  edu.Institution,
			Degree:       edu.Degree,
			Period:       edu.Period,
			DisplayOrder: int32(i),
		})
		if err != nil {
			return fmt.Errorf("creating education %d: %w", i, err)
		}
	}

	// Insert new skill entries
	for i, skill := range sanitized.Skills {
		_, err := s.resumeRepo.CreateSkill(ctx, db.CreateSkillParams{
			ResumeID:     pgResumeID,
			Name:         skill,
			DisplayOrder: int32(i),
		})
		if err != nil {
			return fmt.Errorf("creating skill %d: %w", i, err)
		}
	}

	return nil
}

// Delete verifies ownership and deletes the resume (CASCADE handles children).
func (s *resumeService) Delete(ctx context.Context, userID, resumeID uuid.UUID) error {
	dbResume, err := s.resumeRepo.FindByID(ctx, resumeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.ErrNotFound
		}
		return err
	}

	if err := s.verifyOwnership(dbResume, userID); err != nil {
		return err
	}

	return s.resumeRepo.Delete(ctx, resumeID)
}

// Duplicate verifies ownership, creates a copy with title + " (cópia)", and copies all children.
func (s *resumeService) Duplicate(ctx context.Context, userID, resumeID uuid.UUID) (*models.Resume, error) {
	dbResume, err := s.resumeRepo.FindByID(ctx, resumeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}

	if err := s.verifyOwnership(dbResume, userID); err != nil {
		return nil, err
	}

	// Create the duplicate resume with modified title
	newResume, err := s.resumeRepo.Create(ctx, db.CreateResumeParams{
		UserID:        dbResume.UserID,
		Title:         dbResume.Title + " (cópia)",
		TemplateName:  dbResume.TemplateName,
		PersonalName:  dbResume.PersonalName,
		PersonalTitle: dbResume.PersonalTitle,
		Email:         dbResume.Email,
		Phone:         dbResume.Phone,
		Location:      dbResume.Location,
		Summary:       dbResume.Summary,
		PhotoUrl:      dbResume.PhotoUrl,
	})
	if err != nil {
		return nil, err
	}

	// Copy experiences
	experiences, err := s.resumeRepo.FindExperiencesByResumeID(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	for _, exp := range experiences {
		_, err := s.resumeRepo.CreateExperience(ctx, db.CreateExperienceParams{
			ResumeID:     newResume.ID,
			Company:      exp.Company,
			Role:         exp.Role,
			Period:       exp.Period,
			Description:  exp.Description,
			DisplayOrder: exp.DisplayOrder,
		})
		if err != nil {
			return nil, err
		}
	}

	// Copy educations
	educations, err := s.resumeRepo.FindEducationsByResumeID(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	for _, edu := range educations {
		_, err := s.resumeRepo.CreateEducation(ctx, db.CreateEducationParams{
			ResumeID:     newResume.ID,
			Institution:  edu.Institution,
			Degree:       edu.Degree,
			Period:       edu.Period,
			DisplayOrder: edu.DisplayOrder,
		})
		if err != nil {
			return nil, err
		}
	}

	// Copy skills
	skills, err := s.resumeRepo.FindSkillsByResumeID(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	for _, skill := range skills {
		_, err := s.resumeRepo.CreateSkill(ctx, db.CreateSkillParams{
			ResumeID:     newResume.ID,
			Name:         skill.Name,
			DisplayOrder: skill.DisplayOrder,
		})
		if err != nil {
			return nil, err
		}
	}

	result := dbResumeToModel(newResume)
	return &result, nil
}

// GenerateShareToken verifies ownership, generates a UUID v4 token, and persists it.
func (s *resumeService) GenerateShareToken(ctx context.Context, userID, resumeID uuid.UUID) (string, error) {
	dbResume, err := s.resumeRepo.FindByID(ctx, resumeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", models.ErrNotFound
		}
		return "", err
	}

	if err := s.verifyOwnership(dbResume, userID); err != nil {
		return "", err
	}

	token := uuid.New().String()
	if err := s.resumeRepo.SetShareToken(ctx, resumeID, &token); err != nil {
		return "", err
	}

	return token, nil
}

// RevokeShareToken verifies ownership and sets share_token to nil.
func (s *resumeService) RevokeShareToken(ctx context.Context, userID, resumeID uuid.UUID) error {
	dbResume, err := s.resumeRepo.FindByID(ctx, resumeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.ErrNotFound
		}
		return err
	}

	if err := s.verifyOwnership(dbResume, userID); err != nil {
		return err
	}

	return s.resumeRepo.SetShareToken(ctx, resumeID, nil)
}

// RegenerateShareToken verifies ownership, generates a new UUID v4 token, and replaces the existing one.
func (s *resumeService) RegenerateShareToken(ctx context.Context, userID, resumeID uuid.UUID) (string, error) {
	dbResume, err := s.resumeRepo.FindByID(ctx, resumeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", models.ErrNotFound
		}
		return "", err
	}

	if err := s.verifyOwnership(dbResume, userID); err != nil {
		return "", err
	}

	token := uuid.New().String()
	if err := s.resumeRepo.SetShareToken(ctx, resumeID, &token); err != nil {
		return "", err
	}

	return token, nil
}

// GetByShareToken finds a resume by its share token (public, no auth required) and loads children.
func (s *resumeService) GetByShareToken(ctx context.Context, token string) (*models.ResumeDetail, error) {
	dbResume, err := s.resumeRepo.FindByShareToken(ctx, token)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}

	return s.loadResumeDetail(ctx, dbResume)
}

// UpdatePhotoURL verifies ownership and updates the photo URL for a resume.
func (s *resumeService) UpdatePhotoURL(ctx context.Context, userID, resumeID uuid.UUID, photoURL string) error {
	dbResume, err := s.resumeRepo.FindByID(ctx, resumeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.ErrNotFound
		}
		return err
	}

	if err := s.verifyOwnership(dbResume, userID); err != nil {
		return err
	}

	return s.resumeRepo.UpdatePhotoURL(ctx, resumeID, photoURL)
}

// UpdateThumbnailURL verifies ownership and updates the thumbnail URL for a resume.
func (s *resumeService) UpdateThumbnailURL(ctx context.Context, userID, resumeID uuid.UUID, thumbnailURL string) error {
	dbResume, err := s.resumeRepo.FindByID(ctx, resumeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.ErrNotFound
		}
		return err
	}

	if err := s.verifyOwnership(dbResume, userID); err != nil {
		return err
	}

	return s.resumeRepo.UpdateThumbnailURL(ctx, resumeID, thumbnailURL)
}

// --- Internal helpers ---

// verifyOwnership checks that the resume belongs to the given user.
func (s *resumeService) verifyOwnership(resume db.Resume, userID uuid.UUID) error {
	if resume.UserID.Bytes != userID {
		return models.ErrForbidden
	}
	return nil
}

// loadResumeDetail loads all child entities for a given db.Resume and returns a models.ResumeDetail.
func (s *resumeService) loadResumeDetail(ctx context.Context, dbResume db.Resume) (*models.ResumeDetail, error) {
	resumeID := uuid.UUID(dbResume.ID.Bytes)

	experiences, err := s.resumeRepo.FindExperiencesByResumeID(ctx, resumeID)
	if err != nil {
		return nil, err
	}

	educations, err := s.resumeRepo.FindEducationsByResumeID(ctx, resumeID)
	if err != nil {
		return nil, err
	}

	skills, err := s.resumeRepo.FindSkillsByResumeID(ctx, resumeID)
	if err != nil {
		return nil, err
	}

	detail := &models.ResumeDetail{
		Resume:     dbResumeToModel(dbResume),
		Experience: dbExperiencesToModels(experiences),
		Education:  dbEducationsToModels(educations),
		Skills:     dbSkillsToModels(skills),
	}

	return detail, nil
}

// validateResumeInput validates the resume creation/update input.
func validateResumeInput(input CreateResumeInput) error {
	fields := make(map[string]string)

	// Template must be one of the allowed choices
	if err := validators.ValidateTemplateChoice(input.TemplateName); err != nil {
		if ve, ok := err.(*models.ValidationError); ok {
			for k, v := range ve.Fields {
				fields[k] = v
			}
		}
	}

	// Personal name is required
	if err := validators.ValidateRequired("personal_name", input.PersonalName); err != nil {
		if ve, ok := err.(*models.ValidationError); ok {
			for k, v := range ve.Fields {
				fields[k] = v
			}
		}
	}

	// Field length validations
	if err := validators.ValidateMaxLength("personal_name", input.PersonalName, validators.MaxNameLength); err != nil {
		if ve, ok := err.(*models.ValidationError); ok {
			for k, v := range ve.Fields {
				fields[k] = v
			}
		}
	}

	if err := validators.ValidateMaxLength("title", input.Title, validators.MaxTitleLength); err != nil {
		if ve, ok := err.(*models.ValidationError); ok {
			for k, v := range ve.Fields {
				fields[k] = v
			}
		}
	}

	if err := validators.ValidateMaxLength("personal_title", input.PersonalTitle, validators.MaxTitleLength); err != nil {
		if ve, ok := err.(*models.ValidationError); ok {
			for k, v := range ve.Fields {
				fields[k] = v
			}
		}
	}

	if err := validators.ValidateMaxLength("summary", input.Summary, validators.MaxSummaryLength); err != nil {
		if ve, ok := err.(*models.ValidationError); ok {
			for k, v := range ve.Fields {
				fields[k] = v
			}
		}
	}

	// Validate experience descriptions
	for i, exp := range input.Experience {
		fieldName := fmt.Sprintf("experience[%d].description", i)
		if err := validators.ValidateMaxLength(fieldName, exp.Description, validators.MaxExperienceDescLength); err != nil {
			if ve, ok := err.(*models.ValidationError); ok {
				for k, v := range ve.Fields {
					fields[k] = v
				}
			}
		}
	}

	// Validate skill names
	for i, skill := range input.Skills {
		fieldName := fmt.Sprintf("skills[%d]", i)
		if err := validators.ValidateMaxLength(fieldName, skill, validators.MaxSkillNameLength); err != nil {
			if ve, ok := err.(*models.ValidationError); ok {
				for k, v := range ve.Fields {
					fields[k] = v
				}
			}
		}
	}

	if len(fields) > 0 {
		return &models.ValidationError{Fields: fields}
	}
	return nil
}

// sanitizeResumeInput sanitizes all text fields in the input using TrimAndSanitize.
func sanitizeResumeInput(input CreateResumeInput) CreateResumeInput {
	sanitized := CreateResumeInput{
		Title:         validators.TrimAndSanitize(input.Title),
		TemplateName:  input.TemplateName, // Template name is a controlled value, no sanitization needed
		PersonalName:  validators.TrimAndSanitize(input.PersonalName),
		PersonalTitle: validators.TrimAndSanitize(input.PersonalTitle),
		Email:         validators.TrimAndSanitize(input.Email),
		Phone:         validators.TrimAndSanitize(input.Phone),
		Location:      validators.TrimAndSanitize(input.Location),
		Summary:       validators.TrimAndSanitize(input.Summary),
		PhotoURL:      input.PhotoURL, // PhotoURL is a server-controlled path, no sanitization needed
	}

	// Sanitize experience entries
	sanitized.Experience = make([]ExperienceInput, len(input.Experience))
	for i, exp := range input.Experience {
		sanitized.Experience[i] = ExperienceInput{
			Company:     validators.TrimAndSanitize(exp.Company),
			Role:        validators.TrimAndSanitize(exp.Role),
			Period:      validators.TrimAndSanitize(exp.Period),
			Description: validators.TrimAndSanitize(exp.Description),
		}
	}

	// Sanitize education entries
	sanitized.Education = make([]EducationInput, len(input.Education))
	for i, edu := range input.Education {
		sanitized.Education[i] = EducationInput{
			Institution: validators.TrimAndSanitize(edu.Institution),
			Degree:      validators.TrimAndSanitize(edu.Degree),
			Period:      validators.TrimAndSanitize(edu.Period),
		}
	}

	// Sanitize skill names
	sanitized.Skills = make([]string, len(input.Skills))
	for i, skill := range input.Skills {
		sanitized.Skills[i] = validators.TrimAndSanitize(skill)
	}

	return sanitized
}

// --- DB to domain model converters ---

// dbResumeToModel converts a db.Resume (SQLC-generated) to a models.Resume (domain model).
func dbResumeToModel(r db.Resume) models.Resume {
	m := models.Resume{
		ID:            uuid.UUID(r.ID.Bytes),
		UserID:        uuid.UUID(r.UserID.Bytes),
		Title:         r.Title,
		TemplateName:  r.TemplateName,
		PersonalName:  r.PersonalName,
		PersonalTitle: r.PersonalTitle,
		Email:         r.Email,
		Phone:         r.Phone,
		Location:      r.Location,
		Summary:       r.Summary,
		PhotoURL:      r.PhotoUrl,
		ThumbnailURL:  r.ThumbnailUrl,
		CreatedAt:     r.CreatedAt.Time,
		UpdatedAt:     r.UpdatedAt.Time,
	}

	if r.ShareToken.Valid {
		token := r.ShareToken.String
		m.ShareToken = &token
	}

	return m
}

// dbExperiencesToModels converts a slice of db.Experience to []models.Experience.
func dbExperiencesToModels(exps []db.Experience) []models.Experience {
	result := make([]models.Experience, len(exps))
	for i, e := range exps {
		result[i] = models.Experience{
			ID:           uuid.UUID(e.ID.Bytes),
			ResumeID:     uuid.UUID(e.ResumeID.Bytes),
			Company:      e.Company,
			Role:         e.Role,
			Period:       e.Period,
			Description:  e.Description,
			DisplayOrder: int(e.DisplayOrder),
		}
	}
	return result
}

// dbEducationsToModels converts a slice of db.Education to []models.Education.
func dbEducationsToModels(edus []db.Education) []models.Education {
	result := make([]models.Education, len(edus))
	for i, e := range edus {
		result[i] = models.Education{
			ID:           uuid.UUID(e.ID.Bytes),
			ResumeID:     uuid.UUID(e.ResumeID.Bytes),
			Institution:  e.Institution,
			Degree:       e.Degree,
			Period:       e.Period,
			DisplayOrder: int(e.DisplayOrder),
		}
	}
	return result
}

// dbSkillsToModels converts a slice of db.Skill to []models.Skill.
func dbSkillsToModels(skills []db.Skill) []models.Skill {
	result := make([]models.Skill, len(skills))
	for i, sk := range skills {
		result[i] = models.Skill{
			ID:           uuid.UUID(sk.ID.Bytes),
			ResumeID:     uuid.UUID(sk.ResumeID.Bytes),
			Name:         sk.Name,
			DisplayOrder: int(sk.DisplayOrder),
		}
	}
	return result
}

// uuidToPgtype converts a google/uuid.UUID to pgtype.UUID.
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

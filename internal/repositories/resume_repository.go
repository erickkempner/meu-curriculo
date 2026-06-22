package repositories

import (
	"context"

	"github.com/erick/curriculo/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ResumeRepository defines the persistence operations for resumes and their child entities.
type ResumeRepository interface {
	Create(ctx context.Context, params db.CreateResumeParams) (db.Resume, error)
	FindByID(ctx context.Context, id uuid.UUID) (db.Resume, error)
	FindAllByUserID(ctx context.Context, userID uuid.UUID) ([]db.Resume, error)
	Update(ctx context.Context, params db.UpdateResumeParams) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByShareToken(ctx context.Context, token string) (db.Resume, error)
	SetShareToken(ctx context.Context, id uuid.UUID, token *string) error
	UpdatePhotoURL(ctx context.Context, id uuid.UUID, photoURL string) error
	UpdateThumbnailURL(ctx context.Context, id uuid.UUID, thumbnailURL string) error

	// Experience
	CreateExperience(ctx context.Context, params db.CreateExperienceParams) (db.Experience, error)
	FindExperiencesByResumeID(ctx context.Context, resumeID uuid.UUID) ([]db.Experience, error)
	DeleteExperiencesByResumeID(ctx context.Context, resumeID uuid.UUID) error

	// Education
	CreateEducation(ctx context.Context, params db.CreateEducationParams) (db.Education, error)
	FindEducationsByResumeID(ctx context.Context, resumeID uuid.UUID) ([]db.Education, error)
	DeleteEducationsByResumeID(ctx context.Context, resumeID uuid.UUID) error

	// Skills
	CreateSkill(ctx context.Context, params db.CreateSkillParams) (db.Skill, error)
	FindSkillsByResumeID(ctx context.Context, resumeID uuid.UUID) ([]db.Skill, error)
	DeleteSkillsByResumeID(ctx context.Context, resumeID uuid.UUID) error
}

// resumeRepository implements ResumeRepository using SQLC-generated queries.
type resumeRepository struct {
	queries *db.Queries
}

// NewResumeRepository creates a new ResumeRepository backed by SQLC queries.
func NewResumeRepository(queries *db.Queries) ResumeRepository {
	return &resumeRepository{queries: queries}
}

// uuidToPgtype converts a google/uuid.UUID to pgtype.UUID.
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// stringPtrToPgtext converts a *string to pgtype.Text (nullable).
func stringPtrToPgtext(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// --- Resume CRUD ---

func (r *resumeRepository) Create(ctx context.Context, params db.CreateResumeParams) (db.Resume, error) {
	return r.queries.CreateResume(ctx, params)
}

func (r *resumeRepository) FindByID(ctx context.Context, id uuid.UUID) (db.Resume, error) {
	return r.queries.FindResumeByID(ctx, uuidToPgtype(id))
}

func (r *resumeRepository) FindAllByUserID(ctx context.Context, userID uuid.UUID) ([]db.Resume, error) {
	return r.queries.FindResumesByUserID(ctx, uuidToPgtype(userID))
}

func (r *resumeRepository) Update(ctx context.Context, params db.UpdateResumeParams) error {
	return r.queries.UpdateResume(ctx, params)
}

func (r *resumeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteResume(ctx, uuidToPgtype(id))
}

func (r *resumeRepository) FindByShareToken(ctx context.Context, token string) (db.Resume, error) {
	return r.queries.FindResumeByShareToken(ctx, pgtype.Text{String: token, Valid: true})
}

func (r *resumeRepository) SetShareToken(ctx context.Context, id uuid.UUID, token *string) error {
	return r.queries.SetShareToken(ctx, db.SetShareTokenParams{
		ID:         uuidToPgtype(id),
		ShareToken: stringPtrToPgtext(token),
	})
}

func (r *resumeRepository) UpdatePhotoURL(ctx context.Context, id uuid.UUID, photoURL string) error {
	return r.queries.UpdatePhotoURL(ctx, db.UpdatePhotoURLParams{
		ID:       uuidToPgtype(id),
		PhotoUrl: photoURL,
	})
}

func (r *resumeRepository) UpdateThumbnailURL(ctx context.Context, id uuid.UUID, thumbnailURL string) error {
	return r.queries.UpdateThumbnailURL(ctx, db.UpdateThumbnailURLParams{
		ID:           uuidToPgtype(id),
		ThumbnailUrl: thumbnailURL,
	})
}

// --- Experience ---

func (r *resumeRepository) CreateExperience(ctx context.Context, params db.CreateExperienceParams) (db.Experience, error) {
	return r.queries.CreateExperience(ctx, params)
}

func (r *resumeRepository) FindExperiencesByResumeID(ctx context.Context, resumeID uuid.UUID) ([]db.Experience, error) {
	return r.queries.FindExperiencesByResumeID(ctx, uuidToPgtype(resumeID))
}

func (r *resumeRepository) DeleteExperiencesByResumeID(ctx context.Context, resumeID uuid.UUID) error {
	return r.queries.DeleteExperiencesByResumeID(ctx, uuidToPgtype(resumeID))
}

// --- Education ---

func (r *resumeRepository) CreateEducation(ctx context.Context, params db.CreateEducationParams) (db.Education, error) {
	return r.queries.CreateEducation(ctx, params)
}

func (r *resumeRepository) FindEducationsByResumeID(ctx context.Context, resumeID uuid.UUID) ([]db.Education, error) {
	return r.queries.FindEducationsByResumeID(ctx, uuidToPgtype(resumeID))
}

func (r *resumeRepository) DeleteEducationsByResumeID(ctx context.Context, resumeID uuid.UUID) error {
	return r.queries.DeleteEducationsByResumeID(ctx, uuidToPgtype(resumeID))
}

// --- Skills ---

func (r *resumeRepository) CreateSkill(ctx context.Context, params db.CreateSkillParams) (db.Skill, error) {
	return r.queries.CreateSkill(ctx, params)
}

func (r *resumeRepository) FindSkillsByResumeID(ctx context.Context, resumeID uuid.UUID) ([]db.Skill, error) {
	return r.queries.FindSkillsByResumeID(ctx, uuidToPgtype(resumeID))
}

func (r *resumeRepository) DeleteSkillsByResumeID(ctx context.Context, resumeID uuid.UUID) error {
	return r.queries.DeleteSkillsByResumeID(ctx, uuidToPgtype(resumeID))
}

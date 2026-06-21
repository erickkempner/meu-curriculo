package services

import (
	"context"
	"html"
	"net/mail"
	"strings"

	"github.com/erick/curriculo/internal/db"
	"github.com/erick/curriculo/internal/models"
	"github.com/erick/curriculo/internal/repositories"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost defines the cost factor for password hashing (≥ 12).
const bcryptCost = 12

// RegisterInput holds the data required to register a new user.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// AuthService defines the authentication operations.
type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*models.User, error)
	Login(ctx context.Context, email, password string) (*models.User, error)
}

// authService implements AuthService using a UserRepository.
type authService struct {
	userRepo repositories.UserRepository
}

// NewAuthService creates a new AuthService backed by the given UserRepository.
func NewAuthService(userRepo repositories.UserRepository) AuthService {
	return &authService{userRepo: userRepo}
}

// Register validates input, checks for duplicate email, hashes the password,
// sanitizes the name, and creates the user via the repository.
func (s *authService) Register(ctx context.Context, input RegisterInput) (*models.User, error) {
	// Validate input fields
	if err := validateRegisterInput(input); err != nil {
		return nil, err
	}

	// Check if email already exists
	_, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err == nil {
		// User found — email is already registered
		return nil, models.ErrDuplicateEmail
	}
	if err != pgx.ErrNoRows {
		// Unexpected error from repository
		return nil, err
	}

	// Hash password with bcrypt cost ≥ 12
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcryptCost)
	if err != nil {
		return nil, err
	}

	// Sanitize name before storage
	sanitizedName := html.EscapeString(strings.TrimSpace(input.Name))

	// Create user via repository
	dbUser, err := s.userRepo.Create(ctx, db.CreateUserParams{
		Name:         sanitizedName,
		Email:        strings.ToLower(strings.TrimSpace(input.Email)),
		PasswordHash: string(hashedPassword),
		Provider:     "local",
	})
	if err != nil {
		return nil, err
	}

	// Convert db.User to models.User
	user := dbUserToModel(dbUser)
	return &user, nil
}

// Login finds the user by email and verifies the password.
// Returns ErrInvalidCredentials for both non-existent email and wrong password.
func (s *authService) Login(ctx context.Context, email, password string) (*models.User, error) {
	dbUser, err := s.userRepo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrInvalidCredentials
		}
		return nil, err
	}

	// Compare password with stored bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.PasswordHash), []byte(password)); err != nil {
		return nil, models.ErrInvalidCredentials
	}

	user := dbUserToModel(dbUser)
	return &user, nil
}

// validateRegisterInput checks name, email, and password constraints.
func validateRegisterInput(input RegisterInput) error {
	fields := make(map[string]string)

	// Name: not empty, min 1 char, max 200 chars
	trimmedName := strings.TrimSpace(input.Name)
	if trimmedName == "" {
		fields["name"] = "nome é obrigatório"
	} else if len([]rune(trimmedName)) > 200 {
		fields["name"] = "nome deve ter no máximo 200 caracteres"
	}

	// Email: valid format
	trimmedEmail := strings.TrimSpace(input.Email)
	if trimmedEmail == "" {
		fields["email"] = "e-mail é obrigatório"
	} else if !isValidEmail(trimmedEmail) {
		fields["email"] = "formato de e-mail inválido"
	}

	// Password: min 8, max 72 chars (bcrypt limit)
	if len(input.Password) < 8 {
		fields["password"] = "senha deve ter no mínimo 8 caracteres"
	} else if len(input.Password) > 72 {
		fields["password"] = "senha deve ter no máximo 72 caracteres"
	}

	if len(fields) > 0 {
		return &models.ValidationError{Fields: fields}
	}
	return nil
}

// isValidEmail checks if the email has a valid format using net/mail.
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// dbUserToModel converts a db.User (SQLC-generated) to a models.User (domain model).
func dbUserToModel(u db.User) models.User {
	return models.User{
		ID:           uuid.UUID(u.ID.Bytes),
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Provider:     u.Provider,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

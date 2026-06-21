package services

import (
	"context"
	"errors"
	"testing"

	"github.com/erick/curriculo/internal/db"
	"github.com/erick/curriculo/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepo implements repositories.UserRepository for testing.
type mockUserRepo struct {
	users       map[string]db.User
	createError error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]db.User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	if m.createError != nil {
		return db.User{}, m.createError
	}
	id := uuid.New()
	user := db.User{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		Name:         params.Name,
		Email:        params.Email,
		PasswordHash: params.PasswordHash,
		Provider:     params.Provider,
		CreatedAt:    pgtype.Timestamptz{Valid: true},
		UpdatedAt:    pgtype.Timestamptz{Valid: true},
	}
	m.users[params.Email] = user
	return user, nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (db.User, error) {
	user, ok := m.users[email]
	if !ok {
		return db.User{}, pgx.ErrNoRows
	}
	return user, nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	for _, u := range m.users {
		if u.ID.Bytes == id {
			return u, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}

func TestRegister_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	user, err := svc.Register(context.Background(), RegisterInput{
		Name:     "João Silva",
		Email:    "joao@example.com",
		Password: "senhaforte123",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Email != "joao@example.com" {
		t.Errorf("expected email joao@example.com, got %s", user.Email)
	}
	// Name should be sanitized (no HTML chars in this case, so same)
	if user.Name != "João Silva" {
		t.Errorf("expected name 'João Silva', got %s", user.Name)
	}
	// Password hash should be valid bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("senhaforte123")); err != nil {
		t.Errorf("password hash verification failed: %v", err)
	}
}

func TestRegister_SanitizesName(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	user, err := svc.Register(context.Background(), RegisterInput{
		Name:     "<script>alert('xss')</script>",
		Email:    "test@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
	if user.Name != expected {
		t.Errorf("expected sanitized name %q, got %q", expected, user.Name)
	}
}

func TestRegister_BcryptCostAtLeast12(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	user, err := svc.Register(context.Background(), RegisterInput{
		Name:     "Test User",
		Email:    "cost@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	cost, err := bcrypt.Cost([]byte(user.PasswordHash))
	if err != nil {
		t.Fatalf("failed to get bcrypt cost: %v", err)
	}
	if cost < 12 {
		t.Errorf("expected bcrypt cost >= 12, got %d", cost)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	// Register first time
	_, err := svc.Register(context.Background(), RegisterInput{
		Name:     "User One",
		Email:    "dup@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Register second time with same email
	_, err = svc.Register(context.Background(), RegisterInput{
		Name:     "User Two",
		Email:    "dup@example.com",
		Password: "password456",
	})
	if !errors.Is(err, models.ErrDuplicateEmail) {
		t.Errorf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestRegister_EmptyName(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	_, err := svc.Register(context.Background(), RegisterInput{
		Name:     "",
		Email:    "test@example.com",
		Password: "password123",
	})

	var ve *models.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if _, ok := ve.Fields["name"]; !ok {
		t.Error("expected validation error for 'name' field")
	}
}

func TestRegister_WhitespaceOnlyName(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	_, err := svc.Register(context.Background(), RegisterInput{
		Name:     "   ",
		Email:    "test@example.com",
		Password: "password123",
	})

	var ve *models.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if _, ok := ve.Fields["name"]; !ok {
		t.Error("expected validation error for 'name' field")
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	testCases := []string{
		"notanemail",
		"@missing.com",
		"missing@",
		"",
	}

	for _, email := range testCases {
		_, err := svc.Register(context.Background(), RegisterInput{
			Name:     "Test",
			Email:    email,
			Password: "password123",
		})

		var ve *models.ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("email %q: expected ValidationError, got %v", email, err)
			continue
		}
		if _, ok := ve.Fields["email"]; !ok {
			t.Errorf("email %q: expected validation error for 'email' field", email)
		}
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	_, err := svc.Register(context.Background(), RegisterInput{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "short",
	})

	var ve *models.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if _, ok := ve.Fields["password"]; !ok {
		t.Error("expected validation error for 'password' field")
	}
}

func TestRegister_PasswordTooLong(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	longPassword := string(make([]byte, 73)) // 73 chars exceeds 72 limit
	for i := range longPassword {
		_ = i
	}
	// Create a 73-char password
	pw := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 75 chars

	_, err := svc.Register(context.Background(), RegisterInput{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: pw[:73],
	})

	var ve *models.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if _, ok := ve.Fields["password"]; !ok {
		t.Error("expected validation error for 'password' field")
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	// Register user first
	_, err := svc.Register(context.Background(), RegisterInput{
		Name:     "Login Test",
		Email:    "login@example.com",
		Password: "correctpassword",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Login with correct credentials
	user, err := svc.Login(context.Background(), "login@example.com", "correctpassword")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Email != "login@example.com" {
		t.Errorf("expected email login@example.com, got %s", user.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	// Register user first
	_, err := svc.Register(context.Background(), RegisterInput{
		Name:     "Login Test",
		Email:    "login@example.com",
		Password: "correctpassword",
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Login with wrong password
	_, err = svc.Login(context.Background(), "login@example.com", "wrongpassword")
	if !errors.Is(err, models.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_NonExistentEmail(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	// Login with non-existent email
	_, err := svc.Login(context.Background(), "nobody@example.com", "anypassword")
	if !errors.Is(err, models.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRegister_NameMaxLength(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	// 201 rune name should fail
	longName := make([]rune, 201)
	for i := range longName {
		longName[i] = 'a'
	}

	_, err := svc.Register(context.Background(), RegisterInput{
		Name:     string(longName),
		Email:    "test@example.com",
		Password: "password123",
	})

	var ve *models.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if _, ok := ve.Fields["name"]; !ok {
		t.Error("expected validation error for 'name' field")
	}
}

func TestRegister_NameExactly200Chars(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo)

	// Exactly 200 rune name should succeed
	name200 := make([]rune, 200)
	for i := range name200 {
		name200[i] = 'a'
	}

	user, err := svc.Register(context.Background(), RegisterInput{
		Name:     string(name200),
		Email:    "test200@example.com",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("expected no error for 200-char name, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
}

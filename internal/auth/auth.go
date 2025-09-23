package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "github.com/mattn/go-sqlite3"
)

// Common errors
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenInvalid       = errors.New("invalid token")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
)

// User represents an authenticated user
type User struct {
	ID                int64
	Email             string
	PasswordHash      string
	Name              string
	Role              string
	EmailVerified     bool
	EmailVerifiedAt   *time.Time
	RememberToken     *string
	ResetToken        *string
	ResetTokenExpiry  *time.Time
	LastLoginAt       *time.Time
	LastLoginIP       *string
	FailedAttempts    int
	LockedUntil       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Session represents a user session
type Session struct {
	ID        string
	UserID    int64
	Token     string
	IPAddress string
	UserAgent string
	ExpiresAt time.Time
	CreatedAt time.Time
	LastUsed  time.Time
}

// Authenticator handles user authentication and authorization
type Authenticator struct {
	db                *sql.DB
	sessionDuration   time.Duration
	rememberDuration  time.Duration
	maxFailedAttempts int
	lockoutDuration   time.Duration
	tokenLength       int
}

// NewAuthenticator creates a new authenticator instance
func NewAuthenticator(dbPath string) (*Authenticator, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	auth := &Authenticator{
		db:                db,
		sessionDuration:   24 * time.Hour,
		rememberDuration:  30 * 24 * time.Hour,
		maxFailedAttempts: 5,
		lockoutDuration:   15 * time.Minute,
		tokenLength:       32,
	}

	if err := auth.createTables(); err != nil {
		db.Close()
		return nil, err
	}

	return auth, nil
}

// createTables creates the necessary database tables
func (a *Authenticator) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		name TEXT,
		role TEXT DEFAULT 'user',
		email_verified BOOLEAN DEFAULT 0,
		email_verified_at TIMESTAMP,
		remember_token TEXT,
		reset_token TEXT,
		reset_token_expiry TIMESTAMP,
		last_login_at TIMESTAMP,
		last_login_ip TEXT,
		failed_attempts INTEGER DEFAULT 0,
		locked_until TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_remember_token ON users(remember_token);
	CREATE INDEX IF NOT EXISTS idx_users_reset_token ON users(reset_token);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		token TEXT UNIQUE NOT NULL,
		ip_address TEXT,
		user_agent TEXT,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

	CREATE TABLE IF NOT EXISTS permissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS role_permissions (
		role TEXT NOT NULL,
		permission_id INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (role, permission_id),
		FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role);
	`

	_, err := a.db.Exec(schema)
	return err
}

// Register creates a new user account
func (a *Authenticator) Register(email, password, name string) (*User, error) {
	// Validate email
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, errors.New("invalid email address")
	}

	// Validate password
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	// Check if user exists
	var exists bool
	err := a.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", email).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, ErrUserExists
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	result, err := a.db.Exec(`
		INSERT INTO users (email, password_hash, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, email, string(hash), name, time.Now(), time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, _ := result.LastInsertId()

	return &User{
		ID:        id,
		Email:     email,
		Name:      name,
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// Login authenticates a user and creates a session
func (a *Authenticator) Login(email, password, ipAddress, userAgent string, remember bool) (*Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	// Get user
	user, err := a.GetUserByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if account is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("account is locked until %s", user.LockedUntil.Format(time.RFC3339))
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Increment failed attempts
		a.incrementFailedAttempts(user.ID)
		return nil, ErrInvalidCredentials
	}

	// Reset failed attempts and update last login
	_, err = a.db.Exec(`
		UPDATE users 
		SET failed_attempts = 0, 
		    locked_until = NULL,
		    last_login_at = ?,
		    last_login_ip = ?,
		    updated_at = ?
		WHERE id = ?
	`, time.Now(), ipAddress, time.Now(), user.ID)
	if err != nil {
		log.Printf("Failed to update user login info: %v", err)
	}

	// Create session
	duration := a.sessionDuration
	if remember {
		duration = a.rememberDuration
	}

	return a.CreateSession(user.ID, ipAddress, userAgent, duration)
}

// Logout destroys a session
func (a *Authenticator) Logout(token string) error {
	_, err := a.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

// CreateSession creates a new session for a user
func (a *Authenticator) CreateSession(userID int64, ipAddress, userAgent string, duration time.Duration) (*Session, error) {
	// Generate session ID and token
	sessionID := a.generateToken(16)
	token := a.generateToken(a.tokenLength)
	expiresAt := time.Now().Add(duration)

	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		Token:     token,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	_, err := a.db.Exec(`
		INSERT INTO sessions (id, user_id, token, ip_address, user_agent, expires_at, created_at, last_used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.Token, session.IPAddress, session.UserAgent,
		session.ExpiresAt, session.CreatedAt, session.LastUsed)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// ValidateSession checks if a session token is valid
func (a *Authenticator) ValidateSession(token string) (*Session, *User, error) {
	var session Session
	var user User

	err := a.db.QueryRow(`
		SELECT s.id, s.user_id, s.token, s.ip_address, s.user_agent, s.expires_at, s.created_at, s.last_used,
		       u.id, u.email, u.name, u.role, u.email_verified
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.token = ? AND s.expires_at > ?
	`, token, time.Now()).Scan(
		&session.ID, &session.UserID, &session.Token, &session.IPAddress, &session.UserAgent,
		&session.ExpiresAt, &session.CreatedAt, &session.LastUsed,
		&user.ID, &user.Email, &user.Name, &user.Role, &user.EmailVerified,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrTokenInvalid
		}
		return nil, nil, err
	}

	// Update last used time
	go func() {
		a.db.Exec("UPDATE sessions SET last_used = ? WHERE id = ?", time.Now(), session.ID)
	}()

	return &session, &user, nil
}

// GetUserByEmail retrieves a user by email
func (a *Authenticator) GetUserByEmail(email string) (*User, error) {
	var user User
	err := a.db.QueryRow(`
		SELECT id, email, password_hash, name, role, email_verified, email_verified_at,
		       last_login_at, last_login_ip, failed_attempts, locked_until,
		       created_at, updated_at
		FROM users WHERE email = ?
	`, strings.ToLower(email)).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
		&user.EmailVerified, &user.EmailVerifiedAt,
		&user.LastLoginAt, &user.LastLoginIP, &user.FailedAttempts, &user.LockedUntil,
		&user.CreatedAt, &user.UpdatedAt,
	)
	return &user, err
}

// GetUserByID retrieves a user by ID
func (a *Authenticator) GetUserByID(id int64) (*User, error) {
	var user User
	err := a.db.QueryRow(`
		SELECT id, email, password_hash, name, role, email_verified, email_verified_at,
		       last_login_at, last_login_ip, failed_attempts, locked_until,
		       created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
		&user.EmailVerified, &user.EmailVerifiedAt,
		&user.LastLoginAt, &user.LastLoginIP, &user.FailedAttempts, &user.LockedUntil,
		&user.CreatedAt, &user.UpdatedAt,
	)
	return &user, err
}

// RequestPasswordReset generates a password reset token
func (a *Authenticator) RequestPasswordReset(email string) (string, error) {
	user, err := a.GetUserByEmail(email)
	if err != nil {
		if err == sql.ErrNoRows {
			// Don't reveal if user exists
			return "", nil
		}
		return "", err
	}

	token := a.generateToken(a.tokenLength)
	expiry := time.Now().Add(1 * time.Hour)

	_, err = a.db.Exec(`
		UPDATE users
		SET reset_token = ?, reset_token_expiry = ?, updated_at = ?
		WHERE id = ?
	`, token, expiry, time.Now(), user.ID)

	if err != nil {
		return "", fmt.Errorf("failed to set reset token: %w", err)
	}

	return token, nil
}

// ResetPassword resets a user's password with a valid token
func (a *Authenticator) ResetPassword(token, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	// Find user with valid token
	var userID int64
	err := a.db.QueryRow(`
		SELECT id FROM users
		WHERE reset_token = ? AND reset_token_expiry > ?
	`, token, time.Now()).Scan(&userID)

	if err != nil {
		if err == sql.ErrNoRows {
			return ErrTokenInvalid
		}
		return err
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and clear reset token
	_, err = a.db.Exec(`
		UPDATE users
		SET password_hash = ?, reset_token = NULL, reset_token_expiry = NULL,
		    failed_attempts = 0, locked_until = NULL, updated_at = ?
		WHERE id = ?
	`, string(hash), time.Now(), userID)

	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	// Invalidate all existing sessions for security
	a.InvalidateUserSessions(userID)

	return nil
}

// ChangePassword changes a user's password
func (a *Authenticator) ChangePassword(userID int64, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	// Get current password hash
	var currentHash string
	err := a.db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentHash)
	if err != nil {
		return err
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(currentPassword))
	if err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	_, err = a.db.Exec(`
		UPDATE users
		SET password_hash = ?, updated_at = ?
		WHERE id = ?
	`, string(hash), time.Now(), userID)

	return err
}

// InvalidateUserSessions removes all sessions for a user
func (a *Authenticator) InvalidateUserSessions(userID int64) error {
	_, err := a.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// CleanupExpiredSessions removes expired sessions
func (a *Authenticator) CleanupExpiredSessions() error {
	result, err := a.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	if err != nil {
		return err
	}

	if rows, _ := result.RowsAffected(); rows > 0 {
		log.Printf("Cleaned up %d expired sessions", rows)
	}

	return nil
}

// incrementFailedAttempts increments failed login attempts and locks account if necessary
func (a *Authenticator) incrementFailedAttempts(userID int64) {
	var attempts int
	a.db.QueryRow("SELECT failed_attempts FROM users WHERE id = ?", userID).Scan(&attempts)
	attempts++

	var lockedUntil *time.Time
	if attempts >= a.maxFailedAttempts {
		t := time.Now().Add(a.lockoutDuration)
		lockedUntil = &t
	}

	a.db.Exec(`
		UPDATE users
		SET failed_attempts = ?, locked_until = ?, updated_at = ?
		WHERE id = ?
	`, attempts, lockedUntil, time.Now(), userID)
}

// generateToken generates a secure random token
func (a *Authenticator) generateToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// Authorization methods

// HasRole checks if a user has a specific role
func (a *Authenticator) HasRole(userID int64, role string) bool {
	var userRole string
	err := a.db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
	if err != nil {
		return false
	}
	return userRole == role || userRole == "admin" // Admins have all roles
}

// HasPermission checks if a user has a specific permission
func (a *Authenticator) HasPermission(userID int64, permission string) bool {
	var count int
	err := a.db.QueryRow(`
		SELECT COUNT(*)
		FROM users u
		JOIN role_permissions rp ON u.role = rp.role
		JOIN permissions p ON rp.permission_id = p.id
		WHERE u.id = ? AND p.name = ?
	`, userID, permission).Scan(&count)

	if err != nil {
		return false
	}
	return count > 0
}

// CreatePermission creates a new permission
func (a *Authenticator) CreatePermission(name, description string) error {
	_, err := a.db.Exec(`
		INSERT INTO permissions (name, description)
		VALUES (?, ?)
	`, name, description)
	return err
}

// AssignPermissionToRole assigns a permission to a role
func (a *Authenticator) AssignPermissionToRole(role, permission string) error {
	// Get permission ID
	var permID int64
	err := a.db.QueryRow("SELECT id FROM permissions WHERE name = ?", permission).Scan(&permID)
	if err != nil {
		return fmt.Errorf("permission not found: %w", err)
	}

	_, err = a.db.Exec(`
		INSERT OR IGNORE INTO role_permissions (role, permission_id)
		VALUES (?, ?)
	`, role, permID)
	return err
}

// UpdateUserRole updates a user's role
func (a *Authenticator) UpdateUserRole(userID int64, role string) error {
	_, err := a.db.Exec(`
		UPDATE users
		SET role = ?, updated_at = ?
		WHERE id = ?
	`, role, time.Now(), userID)
	return err
}

// VerifyEmail marks a user's email as verified
func (a *Authenticator) VerifyEmail(userID int64) error {
	now := time.Now()
	_, err := a.db.Exec(`
		UPDATE users
		SET email_verified = 1, email_verified_at = ?, updated_at = ?
		WHERE id = ?
	`, now, now, userID)
	return err
}

// Close closes the database connection
func (a *Authenticator) Close() error {
	return a.db.Close()
}

// Middleware helpers for HTTP handlers

// AuthMiddleware is a middleware function for HTTP handlers
func (a *Authenticator) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from cookie or header
		token := a.getTokenFromRequest(r)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate session
		session, user, err := a.ValidateSession(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), "user", user)
		ctx = context.WithValue(ctx, "session", session)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireRole creates a middleware that requires a specific role
func (a *Authenticator) RequireRole(role string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value("user").(*User)
			if !ok || !a.HasRole(user.ID, role) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

// RequirePermission creates a middleware that requires a specific permission
func (a *Authenticator) RequirePermission(permission string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value("user").(*User)
			if !ok || !a.HasPermission(user.ID, permission) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

// getTokenFromRequest extracts the authentication token from request
func (a *Authenticator) getTokenFromRequest(r *http.Request) string {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" {
		parts := strings.Split(auth, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// Check cookie
	cookie, err := r.Cookie("session_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}

// ComparePassword securely compares two passwords
func ComparePassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// HashPassword generates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// SecureCompare performs a constant-time comparison of two strings
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
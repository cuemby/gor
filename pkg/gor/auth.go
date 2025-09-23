package gor

import (
	"context"
	"net/http"
	"time"
)

// Auth defines the authentication and authorization interface.
// Inspired by Rails 8's built-in authentication with JWT and session support.
type Auth interface {
	// Authentication
	Authenticate(ctx context.Context, credentials Credentials) (*User, error)
	Login(ctx context.Context, w http.ResponseWriter, r *http.Request, user *User) error
	Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) error

	// User management
	Register(ctx context.Context, userData UserRegistration) (*User, error)
	GetUser(ctx context.Context, userID string) (*User, error)
	UpdateUser(ctx context.Context, userID string, updates UserUpdate) (*User, error)
	DeleteUser(ctx context.Context, userID string) error

	// Session management
	CreateSession(ctx context.Context, userID string) (*Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	RefreshSession(ctx context.Context, sessionID string) (*Session, error)
	DestroySession(ctx context.Context, sessionID string) error

	// Token management (JWT)
	GenerateToken(ctx context.Context, user *User, tokenType TokenType) (*Token, error)
	ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error)
	RefreshToken(ctx context.Context, refreshToken string) (*Token, error)
	RevokeToken(ctx context.Context, tokenID string) error

	// Authorization
	Authorize(ctx context.Context, user *User, resource string, action string) bool
	HasRole(ctx context.Context, user *User, role string) bool
	HasPermission(ctx context.Context, user *User, permission string) bool

	// Password management
	HashPassword(password string) (string, error)
	VerifyPassword(password, hash string) bool
	ResetPassword(ctx context.Context, email string) (*PasswordReset, error)
	ConfirmPasswordReset(ctx context.Context, token string, newPassword string) error

	// Account verification
	SendVerificationEmail(ctx context.Context, user *User) error
	VerifyEmail(ctx context.Context, token string) error

	// Multi-factor authentication
	EnableMFA(ctx context.Context, userID string, method MFAMethod) (*MFASetup, error)
	DisableMFA(ctx context.Context, userID string) error
	VerifyMFA(ctx context.Context, userID string, code string) bool

	// Middleware
	RequireAuth() MiddlewareFunc
	RequireRole(roles ...string) MiddlewareFunc
	RequirePermission(permissions ...string) MiddlewareFunc
	OptionalAuth() MiddlewareFunc
}

// User represents a user in the system.
type User struct {
	ID               string                 `json:"id" gor:"primary_key"`
	Email            string                 `json:"email" gor:"unique;not_null"`
	Username         string                 `json:"username" gor:"unique"`
	PasswordHash     string                 `json:"-" gor:"not_null"`
	FirstName        string                 `json:"first_name"`
	LastName         string                 `json:"last_name"`
	Avatar           string                 `json:"avatar"`
	Status           UserStatus             `json:"status"`
	EmailVerified    bool                   `json:"email_verified"`
	EmailVerifiedAt  *time.Time             `json:"email_verified_at"`
	MFAEnabled       bool                   `json:"mfa_enabled"`
	MFASecret        string                 `json:"-"`
	LastLoginAt      *time.Time             `json:"last_login_at"`
	FailedLoginCount int                    `json:"failed_login_count"`
	LockedAt         *time.Time             `json:"locked_at"`
	Roles            []Role                 `json:"roles" gor:"many_to_many"`
	Permissions      []Permission           `json:"permissions" gor:"many_to_many"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

func (u *User) TableName() string { return "users" }

func (u *User) FullName() string {
	if u.FirstName != "" && u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	if u.LastName != "" {
		return u.LastName
	}
	return u.Username
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

func (u *User) IsLocked() bool {
	return u.LockedAt != nil && u.LockedAt.After(time.Now().Add(-24*time.Hour))
}

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusBanned    UserStatus = "banned"
)

// Role represents a user role.
type Role struct {
	ID          string       `json:"id" gor:"primary_key"`
	Name        string       `json:"name" gor:"unique;not_null"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions" gor:"many_to_many"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func (r *Role) TableName() string { return "roles" }

// Permission represents a specific permission.
type Permission struct {
	ID          string    `json:"id" gor:"primary_key"`
	Name        string    `json:"name" gor:"unique;not_null"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p *Permission) TableName() string { return "permissions" }

// Session represents a user session.
type Session struct {
	ID        string                 `json:"id" gor:"primary_key"`
	UserID    string                 `json:"user_id" gor:"not_null;index"`
	Token     string                 `json:"-" gor:"unique;not_null"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Data      map[string]interface{} `json:"data"`
	ExpiresAt time.Time              `json:"expires_at"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

func (s *Session) TableName() string { return "sessions" }

func (s *Session) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}

// Token represents a JWT token.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope,omitempty"`
}

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
	TokenTypeReset   TokenType = "reset"
	TokenTypeVerify  TokenType = "verify"
)

// TokenClaims represents JWT token claims.
type TokenClaims struct {
	UserID      string                 `json:"user_id"`
	Email       string                 `json:"email"`
	Username    string                 `json:"username"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions"`
	TokenType   TokenType              `json:"token_type"`
	Scope       string                 `json:"scope,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IssuedAt    time.Time              `json:"iat"`
	ExpiresAt   time.Time              `json:"exp"`
	NotBefore   time.Time              `json:"nbf"`
	Issuer      string                 `json:"iss"`
	Subject     string                 `json:"sub"`
	Audience    string                 `json:"aud"`
}

// Credentials for authentication.
type Credentials struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	MFACode  string `json:"mfa_code,omitempty"`
}

// UserRegistration data for new user registration.
type UserRegistration struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,alphanum,min=3,max=50"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"max=100"`
	LastName  string `json:"last_name" validate:"max=100"`
}

// UserUpdate data for updating user information.
type UserUpdate struct {
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
	Username  *string `json:"username,omitempty" validate:"omitempty,alphanum,min=3,max=50"`
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,max=100"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,max=100"`
	Avatar    *string `json:"avatar,omitempty"`
	Status    *UserStatus `json:"status,omitempty"`
}

// PasswordReset represents a password reset request.
type PasswordReset struct {
	ID        string    `json:"id" gor:"primary_key"`
	UserID    string    `json:"user_id" gor:"not_null;index"`
	Token     string    `json:"-" gor:"unique;not_null"`
	ExpiresAt time.Time `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (pr *PasswordReset) TableName() string { return "password_resets" }

func (pr *PasswordReset) IsValid() bool {
	return pr.UsedAt == nil && time.Now().Before(pr.ExpiresAt)
}

// Multi-factor authentication
type MFAMethod string

const (
	MFAMethodTOTP MFAMethod = "totp" // Time-based One-Time Password
	MFAMethodSMS  MFAMethod = "sms"  // SMS verification
	MFAMethodEmail MFAMethod = "email" // Email verification
)

type MFASetup struct {
	Secret    string   `json:"secret"`
	QRCode    string   `json:"qr_code"`
	BackupCodes []string `json:"backup_codes"`
}

// OAuth providers
type OAuthProvider interface {
	Name() string
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (*OAuthToken, error)
	GetUser(ctx context.Context, token *OAuthToken) (*OAuthUser, error)
}

type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type OAuthUser struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Provider string `json:"provider"`
}

// Rate limiting for authentication
type AuthRateLimit interface {
	AllowLogin(identifier string) bool // email or IP
	AllowRegistration(ip string) bool
	AllowPasswordReset(email string) bool
	RecordFailedLogin(identifier string)
	RecordSuccessfulLogin(identifier string)
}

// Authentication events
type AuthEvent struct {
	Type      AuthEventType          `json:"type"`
	UserID    string                 `json:"user_id,omitempty"`
	Email     string                 `json:"email,omitempty"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type AuthEventType string

const (
	AuthEventLogin           AuthEventType = "login"
	AuthEventLogout          AuthEventType = "logout"
	AuthEventRegister        AuthEventType = "register"
	AuthEventPasswordReset   AuthEventType = "password_reset"
	AuthEventEmailVerified   AuthEventType = "email_verified"
	AuthEventMFAEnabled      AuthEventType = "mfa_enabled"
	AuthEventMFADisabled     AuthEventType = "mfa_disabled"
	AuthEventAccountLocked   AuthEventType = "account_locked"
	AuthEventAccountUnlocked AuthEventType = "account_unlocked"
)

// Audit logging for security compliance
type AuthAudit interface {
	LogEvent(ctx context.Context, event AuthEvent) error
	GetUserEvents(ctx context.Context, userID string, since time.Time) ([]AuthEvent, error)
	GetSecurityEvents(ctx context.Context, since time.Time) ([]AuthEvent, error)
}

// Password policies
type PasswordPolicy interface {
	Validate(password string) []string // Returns list of errors
	RequiredLength() int
	RequiresNumbers() bool
	RequiresSpecialChars() bool
	RequiresUppercase() bool
	RequiresLowercase() bool
	ForbidsCommonPasswords() bool
}

// Default password policy
type DefaultPasswordPolicy struct{}

func (p DefaultPasswordPolicy) Validate(password string) []string {
	errors := []string{}
	if len(password) < 8 {
		errors = append(errors, "Password must be at least 8 characters long")
	}
	// Additional validation logic...
	return errors
}

func (p DefaultPasswordPolicy) RequiredLength() int { return 8 }
func (p DefaultPasswordPolicy) RequiresNumbers() bool { return true }
func (p DefaultPasswordPolicy) RequiresSpecialChars() bool { return true }
func (p DefaultPasswordPolicy) RequiresUppercase() bool { return true }
func (p DefaultPasswordPolicy) RequiresLowercase() bool { return true }
func (p DefaultPasswordPolicy) ForbidsCommonPasswords() bool { return true }
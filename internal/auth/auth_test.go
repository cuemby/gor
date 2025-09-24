package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const userContextKey contextKey = "user"

// Helper function to create a test authenticator with in-memory SQLite
func setupTestAuth(t *testing.T) *Authenticator {
	// Use temp file for test database
	tmpFile, err := os.CreateTemp("", "auth_test_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp database: %v", err)
	}
	tmpFile.Close()

	// Clean up database file after test
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	auth, err := NewAuthenticator(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create authenticator: %v", err)
	}

	// Clean up authenticator after test
	t.Cleanup(func() {
		auth.Close()
	})

	return auth
}

func TestNewAuthenticator(t *testing.T) {
	t.Run("ValidDatabase", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "auth_test_*.db")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		auth, err := NewAuthenticator(tmpFile.Name())
		if err != nil {
			t.Fatalf("NewAuthenticator() should not return error: %v", err)
		}
		defer auth.Close()

		if auth.db == nil {
			t.Error("NewAuthenticator() should initialize database connection")
		}

		if auth.sessionDuration != 24*time.Hour {
			t.Errorf("NewAuthenticator() sessionDuration = %v, want %v", auth.sessionDuration, 24*time.Hour)
		}

		if auth.rememberDuration != 30*24*time.Hour {
			t.Errorf("NewAuthenticator() rememberDuration = %v, want %v", auth.rememberDuration, 30*24*time.Hour)
		}

		if auth.maxFailedAttempts != 5 {
			t.Errorf("NewAuthenticator() maxFailedAttempts = %v, want 5", auth.maxFailedAttempts)
		}

		if auth.lockoutDuration != 15*time.Minute {
			t.Errorf("NewAuthenticator() lockoutDuration = %v, want %v", auth.lockoutDuration, 15*time.Minute)
		}
	})

	t.Run("InvalidDatabasePath", func(t *testing.T) {
		_, err := NewAuthenticator("/invalid/path/nonexistent.db")
		if err == nil {
			t.Error("NewAuthenticator() should return error for invalid database path")
		}
	})
}

func TestAuthenticator_Register(t *testing.T) {
	auth := setupTestAuth(t)

	t.Run("ValidRegistration", func(t *testing.T) {
		user, err := auth.Register("test@example.com", "password123", "Test User")
		if err != nil {
			t.Fatalf("Register() should not return error: %v", err)
		}

		if user.ID == 0 {
			t.Error("Register() should set user ID")
		}

		if user.Email != "test@example.com" {
			t.Errorf("Register() Email = %v, want test@example.com", user.Email)
		}

		if user.Name != "Test User" {
			t.Errorf("Register() Name = %v, want Test User", user.Name)
		}

		if user.Role != "user" {
			t.Errorf("Register() Role = %v, want user", user.Role)
		}

		if user.CreatedAt.IsZero() {
			t.Error("Register() should set CreatedAt timestamp")
		}
	})

	t.Run("InvalidEmail", func(t *testing.T) {
		_, err := auth.Register("", "password123", "Test User")
		if err == nil {
			t.Error("Register() should return error for empty email")
		}

		_, err = auth.Register("invalid-email", "password123", "Test User")
		if err == nil {
			t.Error("Register() should return error for invalid email format")
		}
	})

	t.Run("WeakPassword", func(t *testing.T) {
		_, err := auth.Register("test2@example.com", "123", "Test User")
		if err == nil {
			t.Error("Register() should return error for weak password")
		}
	})

	t.Run("DuplicateEmail", func(t *testing.T) {
		// First registration should succeed
		_, err := auth.Register("duplicate@example.com", "password123", "First User")
		if err != nil {
			t.Fatalf("First Register() should not return error: %v", err)
		}

		// Second registration with same email should fail
		_, err = auth.Register("duplicate@example.com", "password456", "Second User")
		if err != ErrUserExists {
			t.Errorf("Register() should return ErrUserExists, got %v", err)
		}
	})

	t.Run("EmailNormalization", func(t *testing.T) {
		// Test email normalization (lowercase, trimming)
		user, err := auth.Register("  NORMALIZE@Example.COM  ", "password123", "Test User")
		if err != nil {
			t.Fatalf("Register() should not return error: %v", err)
		}

		if user.Email != "normalize@example.com" {
			t.Errorf("Register() should normalize email, got %v, want normalize@example.com", user.Email)
		}
	})
}

func TestAuthenticator_Login(t *testing.T) {
	auth := setupTestAuth(t)

	// Create test user
	user, err := auth.Register("login@example.com", "password123", "Login User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("ValidLogin", func(t *testing.T) {
		session, err := auth.Login("login@example.com", "password123", "127.0.0.1", "Test-Agent", false)
		if err != nil {
			t.Fatalf("Login() should not return error: %v", err)
		}

		if session.UserID != user.ID {
			t.Errorf("Login() session.UserID = %v, want %v", session.UserID, user.ID)
		}

		if session.IPAddress != "127.0.0.1" {
			t.Errorf("Login() session.IPAddress = %v, want 127.0.0.1", session.IPAddress)
		}

		if session.UserAgent != "Test-Agent" {
			t.Errorf("Login() session.UserAgent = %v, want Test-Agent", session.UserAgent)
		}

		if session.Token == "" {
			t.Error("Login() should generate session token")
		}

		if session.ExpiresAt.Before(time.Now()) {
			t.Error("Login() session should not be expired")
		}
	})

	t.Run("RememberMe", func(t *testing.T) {
		session, err := auth.Login("login@example.com", "password123", "127.0.0.1", "Test-Agent", true)
		if err != nil {
			t.Fatalf("Login() with remember should not return error: %v", err)
		}

		// Remember me sessions should last longer
		expectedExpiry := time.Now().Add(auth.rememberDuration)
		if session.ExpiresAt.Before(expectedExpiry.Add(-time.Minute)) {
			t.Error("Login() with remember should create longer-lasting session")
		}
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		_, err := auth.Login("login@example.com", "wrongpassword", "127.0.0.1", "Test-Agent", false)
		if err != ErrInvalidCredentials {
			t.Errorf("Login() should return ErrInvalidCredentials, got %v", err)
		}

		_, err = auth.Login("nonexistent@example.com", "password123", "127.0.0.1", "Test-Agent", false)
		if err != ErrInvalidCredentials {
			t.Errorf("Login() should return ErrInvalidCredentials for nonexistent user, got %v", err)
		}
	})

	t.Run("AccountLockout", func(t *testing.T) {
		// Create a fresh user for lockout testing
		lockoutUser, err := auth.Register("lockout@example.com", "password123", "Lockout User")
		if err != nil {
			t.Fatalf("Failed to create lockout test user: %v", err)
		}

		// Try to login with wrong password multiple times
		for i := 0; i < 5; i++ {
			_, err := auth.Login("lockout@example.com", "wrongpassword", "127.0.0.1", "Test-Agent", false)
			if err != ErrInvalidCredentials {
				t.Errorf("Login attempt %d should return ErrInvalidCredentials, got %v", i+1, err)
			}
		}

		// Next login attempt should be locked
		_, err = auth.Login("lockout@example.com", "password123", "127.0.0.1", "Test-Agent", false)
		if err == nil || !strings.Contains(err.Error(), "locked") {
			t.Errorf("Login() should return lockout error, got %v", err)
		}

		// Verify user is actually locked in database
		retrievedUser, _ := auth.GetUserByID(lockoutUser.ID)
		if retrievedUser.LockedUntil == nil || retrievedUser.LockedUntil.Before(time.Now()) {
			t.Error("User should be locked after maximum failed attempts")
		}
	})
}

func TestAuthenticator_SessionManagement(t *testing.T) {
	auth := setupTestAuth(t)

	// Create test user
	user, err := auth.Register("session@example.com", "password123", "Session User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("CreateSession", func(t *testing.T) {
		session, err := auth.CreateSession(user.ID, "127.0.0.1", "Test-Agent", 1*time.Hour)
		if err != nil {
			t.Fatalf("CreateSession() should not return error: %v", err)
		}

		if session.UserID != user.ID {
			t.Errorf("CreateSession() UserID = %v, want %v", session.UserID, user.ID)
		}

		if session.ID == "" {
			t.Error("CreateSession() should generate session ID")
		}

		if session.Token == "" {
			t.Error("CreateSession() should generate token")
		}
	})

	t.Run("ValidateSession", func(t *testing.T) {
		// Create a session
		session, err := auth.CreateSession(user.ID, "127.0.0.1", "Test-Agent", 1*time.Hour)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Validate the session
		validatedSession, validatedUser, err := auth.ValidateSession(session.Token)
		if err != nil {
			t.Fatalf("ValidateSession() should not return error: %v", err)
		}

		if validatedSession.UserID != user.ID {
			t.Errorf("ValidateSession() UserID = %v, want %v", validatedSession.UserID, user.ID)
		}

		if validatedUser.ID != user.ID {
			t.Errorf("ValidateSession() User.ID = %v, want %v", validatedUser.ID, user.ID)
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		_, _, err := auth.ValidateSession("invalid_token")
		if err != ErrTokenInvalid {
			t.Errorf("ValidateSession() should return ErrTokenInvalid, got %v", err)
		}
	})

	t.Run("ExpiredSession", func(t *testing.T) {
		// Create an expired session
		session, err := auth.CreateSession(user.ID, "127.0.0.1", "Test-Agent", -1*time.Hour)
		if err != nil {
			t.Fatalf("Failed to create expired session: %v", err)
		}

		// Validation should fail
		_, _, err = auth.ValidateSession(session.Token)
		if err != ErrTokenInvalid {
			t.Errorf("ValidateSession() should return ErrTokenInvalid for expired session, got %v", err)
		}
	})

	t.Run("Logout", func(t *testing.T) {
		// Create and validate a session
		session, err := auth.CreateSession(user.ID, "127.0.0.1", "Test-Agent", 1*time.Hour)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Session should be valid initially
		_, _, err = auth.ValidateSession(session.Token)
		if err != nil {
			t.Fatalf("Session should be valid before logout: %v", err)
		}

		// Logout
		err = auth.Logout(session.Token)
		if err != nil {
			t.Fatalf("Logout() should not return error: %v", err)
		}

		// Session should be invalid after logout
		_, _, err = auth.ValidateSession(session.Token)
		if err != ErrTokenInvalid {
			t.Error("Session should be invalid after logout")
		}
	})
}

func TestAuthenticator_PasswordReset(t *testing.T) {
	auth := setupTestAuth(t)

	// Create test user
	user, err := auth.Register("reset@example.com", "password123", "Reset User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("RequestPasswordReset", func(t *testing.T) {
		token, err := auth.RequestPasswordReset("reset@example.com")
		if err != nil {
			t.Fatalf("RequestPasswordReset() should not return error: %v", err)
		}

		if token == "" {
			t.Error("RequestPasswordReset() should return reset token")
		}

		// Test that we can use the token later (the actual storage verification happens in ResetPassword test)
		if len(token) < 16 {
			t.Errorf("Reset token should be sufficiently long, got length %d", len(token))
		}
	})

	t.Run("RequestPasswordResetNonexistentUser", func(t *testing.T) {
		// Should not reveal if user exists
		token, err := auth.RequestPasswordReset("nonexistent@example.com")
		if err != nil {
			t.Fatalf("RequestPasswordReset() should not return error for nonexistent user: %v", err)
		}

		if token != "" {
			t.Error("RequestPasswordReset() should return empty token for nonexistent user")
		}
	})

	t.Run("ResetPassword", func(t *testing.T) {
		// Request reset token
		token, err := auth.RequestPasswordReset("reset@example.com")
		if err != nil {
			t.Fatalf("Failed to request reset token: %v", err)
		}

		// Reset password
		err = auth.ResetPassword(token, "newpassword123")
		if err != nil {
			t.Fatalf("ResetPassword() should not return error: %v", err)
		}

		// Verify old password doesn't work
		_, err = auth.Login("reset@example.com", "password123", "127.0.0.1", "Test-Agent", false)
		if err != ErrInvalidCredentials {
			t.Error("Old password should no longer work")
		}

		// Verify new password works
		_, err = auth.Login("reset@example.com", "newpassword123", "127.0.0.1", "Test-Agent", false)
		if err != nil {
			t.Errorf("New password should work: %v", err)
		}

		// Verify reset token is cleared
		retrievedUser, _ := auth.GetUserByID(user.ID)
		if retrievedUser.ResetToken != nil {
			t.Error("Reset token should be cleared after use")
		}
	})

	t.Run("ResetPasswordInvalidToken", func(t *testing.T) {
		err := auth.ResetPassword("invalid_token", "newpassword123")
		if err != ErrTokenInvalid {
			t.Errorf("ResetPassword() should return ErrTokenInvalid, got %v", err)
		}
	})

	t.Run("ResetPasswordWeakPassword", func(t *testing.T) {
		token, _ := auth.RequestPasswordReset("reset@example.com")
		err := auth.ResetPassword(token, "weak")
		if err == nil {
			t.Error("ResetPassword() should reject weak passwords")
		}
	})
}

func TestAuthenticator_ChangePassword(t *testing.T) {
	auth := setupTestAuth(t)

	// Create test user
	user, err := auth.Register("change@example.com", "password123", "Change User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("ValidPasswordChange", func(t *testing.T) {
		err := auth.ChangePassword(user.ID, "password123", "newpassword456")
		if err != nil {
			t.Fatalf("ChangePassword() should not return error: %v", err)
		}

		// Verify old password doesn't work
		_, err = auth.Login("change@example.com", "password123", "127.0.0.1", "Test-Agent", false)
		if err != ErrInvalidCredentials {
			t.Error("Old password should no longer work")
		}

		// Verify new password works
		_, err = auth.Login("change@example.com", "newpassword456", "127.0.0.1", "Test-Agent", false)
		if err != nil {
			t.Errorf("New password should work: %v", err)
		}
	})

	t.Run("WrongCurrentPassword", func(t *testing.T) {
		err := auth.ChangePassword(user.ID, "wrongpassword", "newpassword789")
		if err != ErrInvalidCredentials {
			t.Errorf("ChangePassword() should return ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("WeakNewPassword", func(t *testing.T) {
		err := auth.ChangePassword(user.ID, "newpassword456", "weak")
		if err == nil {
			t.Error("ChangePassword() should reject weak passwords")
		}
	})
}

func TestAuthenticator_Authorization(t *testing.T) {
	auth := setupTestAuth(t)

	// Create test user
	user, err := auth.Register("auth@example.com", "password123", "Auth User")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("HasRole", func(t *testing.T) {
		// User should have default "user" role
		if !auth.HasRole(user.ID, "user") {
			t.Error("User should have 'user' role by default")
		}

		// User should not have admin role
		if auth.HasRole(user.ID, "admin") {
			t.Error("User should not have 'admin' role by default")
		}
	})

	t.Run("UpdateUserRole", func(t *testing.T) {
		err := auth.UpdateUserRole(user.ID, "admin")
		if err != nil {
			t.Fatalf("UpdateUserRole() should not return error: %v", err)
		}

		// User should now have admin role
		if !auth.HasRole(user.ID, "admin") {
			t.Error("User should have 'admin' role after update")
		}

		// Admin should also have user role (as per the HasRole logic)
		if !auth.HasRole(user.ID, "user") {
			t.Error("Admin should also have access to 'user' role")
		}
	})

	t.Run("Permissions", func(t *testing.T) {
		// Create a permission
		err := auth.CreatePermission("read_posts", "Can read blog posts")
		if err != nil {
			t.Fatalf("CreatePermission() should not return error: %v", err)
		}

		// User should not have permission initially
		if auth.HasPermission(user.ID, "read_posts") {
			t.Error("User should not have permission initially")
		}

		// Assign permission to user role
		err = auth.AssignPermissionToRole("user", "read_posts")
		if err != nil {
			t.Fatalf("AssignPermissionToRole() should not return error: %v", err)
		}

		// Update user back to user role for permission test
		_ = auth.UpdateUserRole(user.ID, "user")

		// User should now have permission
		if !auth.HasPermission(user.ID, "read_posts") {
			t.Error("User should have permission after role assignment")
		}
	})
}

func TestAuthenticator_UtilityFunctions(t *testing.T) {
	auth := setupTestAuth(t)

	t.Run("GenerateToken", func(t *testing.T) {
		token1 := auth.generateToken(16)
		token2 := auth.generateToken(16)

		if token1 == "" {
			t.Error("generateToken() should not return empty string")
		}

		if token1 == token2 {
			t.Error("generateToken() should generate unique tokens")
		}

		// Tokens should be base64 URL encoded
		if strings.Contains(token1, "+") || strings.Contains(token1, "/") {
			t.Error("generateToken() should use URL-safe base64 encoding")
		}
	})

	t.Run("CleanupExpiredSessions", func(t *testing.T) {
		// Create test user
		user, _ := auth.Register("cleanup@example.com", "password123", "Cleanup User")

		// Create expired session
		_, _ = auth.CreateSession(user.ID, "127.0.0.1", "Test-Agent", -1*time.Hour)

		// Create valid session
		validSession, _ := auth.CreateSession(user.ID, "127.0.0.1", "Test-Agent", 1*time.Hour)

		// Cleanup expired sessions
		err := auth.CleanupExpiredSessions()
		if err != nil {
			t.Fatalf("CleanupExpiredSessions() should not return error: %v", err)
		}

		// Valid session should still work
		_, _, err = auth.ValidateSession(validSession.Token)
		if err != nil {
			t.Error("Valid session should still work after cleanup")
		}
	})

	t.Run("InvalidateUserSessions", func(t *testing.T) {
		// Create test user
		user, _ := auth.Register("invalidate@example.com", "password123", "Invalidate User")

		// Create multiple sessions
		session1, _ := auth.CreateSession(user.ID, "127.0.0.1", "Test-Agent", 1*time.Hour)
		session2, _ := auth.CreateSession(user.ID, "127.0.0.1", "Test-Agent", 1*time.Hour)

		// Both sessions should be valid initially
		_, _, err := auth.ValidateSession(session1.Token)
		if err != nil {
			t.Fatal("Session1 should be valid initially")
		}
		_, _, err = auth.ValidateSession(session2.Token)
		if err != nil {
			t.Fatal("Session2 should be valid initially")
		}

		// Invalidate all user sessions
		err = auth.InvalidateUserSessions(user.ID)
		if err != nil {
			t.Fatalf("InvalidateUserSessions() should not return error: %v", err)
		}

		// Both sessions should be invalid now
		_, _, err = auth.ValidateSession(session1.Token)
		if err != ErrTokenInvalid {
			t.Error("Session1 should be invalid after invalidation")
		}
		_, _, err = auth.ValidateSession(session2.Token)
		if err != ErrTokenInvalid {
			t.Error("Session2 should be invalid after invalidation")
		}
	})

	t.Run("VerifyEmail", func(t *testing.T) {
		// Create test user
		user, _ := auth.Register("verify@example.com", "password123", "Verify User")

		// User should not be verified initially
		retrievedUser, _ := auth.GetUserByID(user.ID)
		if retrievedUser.EmailVerified {
			t.Error("User should not be verified initially")
		}

		// Verify email
		err := auth.VerifyEmail(user.ID)
		if err != nil {
			t.Fatalf("VerifyEmail() should not return error: %v", err)
		}

		// User should be verified now
		retrievedUser, _ = auth.GetUserByID(user.ID)
		if !retrievedUser.EmailVerified {
			t.Error("User should be verified after VerifyEmail()")
		}

		if retrievedUser.EmailVerifiedAt == nil {
			t.Error("EmailVerifiedAt should be set")
		}
	})
}

func TestAuthenticator_Middleware(t *testing.T) {
	auth := setupTestAuth(t)

	// Create test user and session
	user, _ := auth.Register("middleware@example.com", "password123", "Middleware User")
	session, _ := auth.CreateSession(user.ID, "127.0.0.1", "Test-Agent", 1*time.Hour)

	t.Run("AuthMiddleware_ValidToken", func(t *testing.T) {
		// Create test handler
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			// Check if user is in context
			contextUser, ok := r.Context().Value(UserContextKey).(*User)
			if !ok {
				t.Error("User should be in request context")
				return
			}
			if contextUser.ID != user.ID {
				t.Errorf("Context user ID = %v, want %v", contextUser.ID, user.ID)
			}

			// Check if session is in context
			contextSession, ok := r.Context().Value(SessionContextKey).(*Session)
			if !ok {
				t.Error("Session should be in request context")
				return
			}
			if contextSession.Token != session.Token {
				t.Errorf("Context session token = %v, want %v", contextSession.Token, session.Token)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("authorized"))
		}

		// Wrap with auth middleware
		wrappedHandler := auth.AuthMiddleware(testHandler)

		// Create request with valid token
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+session.Token)
		w := httptest.NewRecorder()

		// Execute request
		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("AuthMiddleware should allow valid token, got status %v", w.Code)
		}

		if w.Body.String() != "authorized" {
			t.Errorf("AuthMiddleware response = %v, want authorized", w.Body.String())
		}
	})

	t.Run("AuthMiddleware_CookieToken", func(t *testing.T) {
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		wrappedHandler := auth.AuthMiddleware(testHandler)

		// Create request with token in cookie
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session_token",
			Value: session.Token,
		})
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("AuthMiddleware should accept token from cookie, got status %v", w.Code)
		}
	})

	t.Run("AuthMiddleware_NoToken", func(t *testing.T) {
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		wrappedHandler := auth.AuthMiddleware(testHandler)

		// Create request without token
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("AuthMiddleware should reject request without token, got status %v", w.Code)
		}
	})

	t.Run("AuthMiddleware_InvalidToken", func(t *testing.T) {
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		wrappedHandler := auth.AuthMiddleware(testHandler)

		// Create request with invalid token
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid_token")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("AuthMiddleware should reject invalid token, got status %v", w.Code)
		}
	})

	t.Run("RequireRole_ValidRole", func(t *testing.T) {
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("role_authorized"))
		}

		roleMiddleware := auth.RequireRole("user")
		wrappedHandler := roleMiddleware(testHandler)

		// Create request with user in context
		req := httptest.NewRequest("GET", "/admin", nil)
		ctx := context.WithValue(req.Context(), UserContextKey, user)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("RequireRole should allow user with correct role, got status %v", w.Code)
		}
	})

	t.Run("RequireRole_InvalidRole", func(t *testing.T) {
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		roleMiddleware := auth.RequireRole("admin")
		wrappedHandler := roleMiddleware(testHandler)

		// Create request with user in context (user role, not admin)
		req := httptest.NewRequest("GET", "/admin", nil)
		ctx := context.WithValue(req.Context(), userContextKey, user)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("RequireRole should reject user without correct role, got status %v", w.Code)
		}
	})
}

func TestPasswordUtilityFunctions(t *testing.T) {
	t.Run("HashPassword", func(t *testing.T) {
		password := "test123"
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword() should not return error: %v", err)
		}

		if hash == "" {
			t.Error("HashPassword() should not return empty string")
		}

		if hash == password {
			t.Error("HashPassword() should not return plain text password")
		}

		// Verify hash can be used with bcrypt
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err != nil {
			t.Error("HashPassword() should generate valid bcrypt hash")
		}
	})

	t.Run("ComparePassword", func(t *testing.T) {
		password := "test123"
		hash, _ := HashPassword(password)

		if !ComparePassword(hash, password) {
			t.Error("ComparePassword() should return true for correct password")
		}

		if ComparePassword(hash, "wrong_password") {
			t.Error("ComparePassword() should return false for incorrect password")
		}
	})

	t.Run("SecureCompare", func(t *testing.T) {
		if !SecureCompare("same", "same") {
			t.Error("SecureCompare() should return true for identical strings")
		}

		if SecureCompare("different", "strings") {
			t.Error("SecureCompare() should return false for different strings")
		}

		// Test timing attack resistance (basic check - just ensure function works)
		if SecureCompare("", "nonempty") {
			t.Error("SecureCompare() should return false for different length strings")
		}
	})
}

func TestAuthenticator_GetUserMethods(t *testing.T) {
	auth := setupTestAuth(t)

	// Create test user
	user, _ := auth.Register("getuser@example.com", "password123", "Get User")

	t.Run("GetUserByEmail", func(t *testing.T) {
		retrievedUser, err := auth.GetUserByEmail("getuser@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail() should not return error: %v", err)
		}

		if retrievedUser.ID != user.ID {
			t.Errorf("GetUserByEmail() ID = %v, want %v", retrievedUser.ID, user.ID)
		}

		if retrievedUser.Email != user.Email {
			t.Errorf("GetUserByEmail() Email = %v, want %v", retrievedUser.Email, user.Email)
		}
	})

	t.Run("GetUserByEmail_NotFound", func(t *testing.T) {
		_, err := auth.GetUserByEmail("nonexistent@example.com")
		if err != sql.ErrNoRows {
			t.Errorf("GetUserByEmail() should return sql.ErrNoRows for nonexistent user, got %v", err)
		}
	})

	t.Run("GetUserByID", func(t *testing.T) {
		retrievedUser, err := auth.GetUserByID(user.ID)
		if err != nil {
			t.Fatalf("GetUserByID() should not return error: %v", err)
		}

		if retrievedUser.ID != user.ID {
			t.Errorf("GetUserByID() ID = %v, want %v", retrievedUser.ID, user.ID)
		}

		if retrievedUser.Email != user.Email {
			t.Errorf("GetUserByID() Email = %v, want %v", retrievedUser.Email, user.Email)
		}
	})

	t.Run("GetUserByID_NotFound", func(t *testing.T) {
		_, err := auth.GetUserByID(99999)
		if err != sql.ErrNoRows {
			t.Errorf("GetUserByID() should return sql.ErrNoRows for nonexistent user, got %v", err)
		}
	})
}

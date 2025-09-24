package gor_test

import (
	"testing"
	"time"

	"github.com/cuemby/gor/pkg/gor"
)

// TestUser tests the User struct and its methods
func TestUser(t *testing.T) {
	user := &gor.User{
		ID:           "user123",
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		FirstName:    "John",
		LastName:     "Doe",
		Status:       gor.UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	t.Run("TableName", func(t *testing.T) {
		tableName := user.TableName()
		expected := "users"
		if tableName != expected {
			t.Errorf("TableName() should return '%s', got '%s'", expected, tableName)
		}
	})

	t.Run("FullName", func(t *testing.T) {
		fullName := user.FullName()
		expected := "John Doe"
		if fullName != expected {
			t.Errorf("FullName() should return '%s', got '%s'", expected, fullName)
		}

		// Test with empty names (should return username)
		user.FirstName = ""
		user.LastName = ""
		fullName = user.FullName()
		expected = "testuser"
		if fullName != expected {
			t.Errorf("FullName() with empty names should return username '%s', got '%s'", expected, fullName)
		}

		// Test with only first name
		user.FirstName = "John"
		user.LastName = ""
		fullName = user.FullName()
		expected = "John"
		if fullName != expected {
			t.Errorf("FullName() with only first name should return '%s', got '%s'", expected, fullName)
		}

		// Test with only last name
		user.FirstName = ""
		user.LastName = "Doe"
		fullName = user.FullName()
		expected = "Doe"
		if fullName != expected {
			t.Errorf("FullName() with only last name should return '%s', got '%s'", expected, fullName)
		}
	})

	t.Run("IsActive", func(t *testing.T) {
		user.Status = gor.UserStatusActive
		if !user.IsActive() {
			t.Error("User should be active when status is active")
		}

		user.Status = gor.UserStatusInactive
		if user.IsActive() {
			t.Error("User should not be active when status is inactive")
		}

		user.Status = gor.UserStatusSuspended
		if user.IsActive() {
			t.Error("User should not be active when status is suspended")
		}
	})

	t.Run("IsLocked", func(t *testing.T) {
		user.LockedAt = nil
		if user.IsLocked() {
			t.Error("User should not be locked when LockedAt is nil")
		}

		// Test locked within 24 hours (should be locked)
		recent := time.Now().Add(-12 * time.Hour)
		user.LockedAt = &recent
		if !user.IsLocked() {
			t.Error("User should be locked when locked recently")
		}

		// Test locked more than 24 hours ago (should not be locked)
		old := time.Now().Add(-25 * time.Hour)
		user.LockedAt = &old
		if user.IsLocked() {
			t.Error("User should not be locked when locked more than 24 hours ago")
		}
	})
}

// TestRole tests the Role struct and its methods
func TestRole(t *testing.T) {
	role := &gor.Role{
		ID:          "role123",
		Name:        "admin",
		Description: "Full system access",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	t.Run("TableName", func(t *testing.T) {
		tableName := role.TableName()
		expected := "roles"
		if tableName != expected {
			t.Errorf("TableName() should return '%s', got '%s'", expected, tableName)
		}
	})

	t.Run("Fields", func(t *testing.T) {
		if role.ID != "role123" {
			t.Errorf("ID should be role123, got %s", role.ID)
		}
		if role.Name != "admin" {
			t.Errorf("Name should be admin, got %s", role.Name)
		}
		if role.Description != "Full system access" {
			t.Errorf("Description should be 'Full system access', got %s", role.Description)
		}
	})
}

// TestPermission tests the Permission struct and its methods
func TestPermission(t *testing.T) {
	permission := &gor.Permission{
		ID:          "perm123",
		Name:        "read_users",
		Resource:    "users",
		Action:      "read",
		Description: "Can read user information",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	t.Run("TableName", func(t *testing.T) {
		tableName := permission.TableName()
		expected := "permissions"
		if tableName != expected {
			t.Errorf("TableName() should return '%s', got '%s'", expected, tableName)
		}
	})

	t.Run("Fields", func(t *testing.T) {
		if permission.ID != "perm123" {
			t.Errorf("ID should be perm123, got %s", permission.ID)
		}
		if permission.Name != "read_users" {
			t.Errorf("Name should be read_users, got %s", permission.Name)
		}
		if permission.Resource != "users" {
			t.Errorf("Resource should be users, got %s", permission.Resource)
		}
		if permission.Action != "read" {
			t.Errorf("Action should be read, got %s", permission.Action)
		}
		if permission.Description != "Can read user information" {
			t.Errorf("Description should be 'Can read user information', got %s", permission.Description)
		}
	})
}

// TestSession tests the Session struct and its methods
func TestSession(t *testing.T) {
	session := &gor.Session{
		ID:        "session123",
		UserID:    "user123",
		Token:     "token123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	t.Run("TableName", func(t *testing.T) {
		tableName := session.TableName()
		expected := "sessions"
		if tableName != expected {
			t.Errorf("TableName() should return '%s', got '%s'", expected, tableName)
		}
	})

	t.Run("IsValid", func(t *testing.T) {
		// Test valid session (expires in the future)
		session.ExpiresAt = time.Now().Add(1 * time.Hour)
		if !session.IsValid() {
			t.Error("Session should be valid when it expires in the future")
		}

		// Test expired session
		session.ExpiresAt = time.Now().Add(-1 * time.Hour)
		if session.IsValid() {
			t.Error("Session should be invalid when it has expired")
		}

		// Test session expiring exactly now (edge case)
		session.ExpiresAt = time.Now()
		// This might be flaky due to timing, but generally should be invalid
		time.Sleep(1 * time.Millisecond)
		if session.IsValid() {
			t.Error("Session should be invalid when it has expired")
		}
	})

	t.Run("Fields", func(t *testing.T) {
		if session.ID != "session123" {
			t.Errorf("ID should be session123, got %s", session.ID)
		}
		if session.UserID != "user123" {
			t.Errorf("UserID should be user123, got %s", session.UserID)
		}
		if session.Token != "token123" {
			t.Errorf("Token should be token123, got %s", session.Token)
		}
	})
}

// TestOAuthToken tests the OAuthToken struct and its methods
func TestOAuthToken(t *testing.T) {
	token := &gor.OAuthToken{
		AccessToken:  "access_token_123",
		RefreshToken: "refresh_token_123",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	t.Run("Fields", func(t *testing.T) {
		if token.AccessToken != "access_token_123" {
			t.Errorf("AccessToken should be access_token_123, got %s", token.AccessToken)
		}
		if token.RefreshToken != "refresh_token_123" {
			t.Errorf("RefreshToken should be refresh_token_123, got %s", token.RefreshToken)
		}
		if token.TokenType != "Bearer" {
			t.Errorf("TokenType should be Bearer, got %s", token.TokenType)
		}
	})

	t.Run("ExpiresAt", func(t *testing.T) {
		// Test that ExpiresAt is in the future
		if !token.ExpiresAt.After(time.Now()) {
			t.Error("Token ExpiresAt should be in the future")
		}
	})
}

// TestDefaultPasswordPolicy tests the DefaultPasswordPolicy implementation
func TestDefaultPasswordPolicy(t *testing.T) {
	policy := gor.DefaultPasswordPolicy{}

	t.Run("Validate_ValidPassword", func(t *testing.T) {
		validPassword := "SecurePass123!"
		errors := policy.Validate(validPassword)
		if len(errors) != 0 {
			t.Errorf("Validate() should accept valid password, got errors: %v", errors)
		}
	})

	t.Run("Validate_TooShort", func(t *testing.T) {
		shortPassword := "Short1!"
		errors := policy.Validate(shortPassword)
		if len(errors) == 0 {
			t.Error("Validate() should reject password shorter than minimum length")
		}
	})

	t.Run("RequiredLength", func(t *testing.T) {
		if policy.RequiredLength() != 8 {
			t.Errorf("RequiredLength() should return 8, got %d", policy.RequiredLength())
		}
	})

	t.Run("RequiresNumbers", func(t *testing.T) {
		if !policy.RequiresNumbers() {
			t.Error("RequiresNumbers() should return true")
		}
	})

	t.Run("RequiresSpecialChars", func(t *testing.T) {
		if !policy.RequiresSpecialChars() {
			t.Error("RequiresSpecialChars() should return true")
		}
	})

	t.Run("RequiresUppercase", func(t *testing.T) {
		if !policy.RequiresUppercase() {
			t.Error("RequiresUppercase() should return true")
		}
	})

	t.Run("RequiresLowercase", func(t *testing.T) {
		if !policy.RequiresLowercase() {
			t.Error("RequiresLowercase() should return true")
		}
	})

	t.Run("ForbidsCommonPasswords", func(t *testing.T) {
		if !policy.ForbidsCommonPasswords() {
			t.Error("ForbidsCommonPasswords() should return true")
		}
	})
}

// TestPasswordReset tests the PasswordReset struct
func TestPasswordReset(t *testing.T) {
	reset := &gor.PasswordReset{
		ID:        "reset123",
		UserID:    "user123",
		Token:     "reset_token_123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	t.Run("TableName", func(t *testing.T) {
		tableName := reset.TableName()
		expected := "password_resets"
		if tableName != expected {
			t.Errorf("TableName() should return '%s', got '%s'", expected, tableName)
		}
	})

	t.Run("IsValid_Valid", func(t *testing.T) {
		reset.UsedAt = nil
		reset.ExpiresAt = time.Now().Add(1 * time.Hour)
		if !reset.IsValid() {
			t.Error("PasswordReset should be valid when not used and not expired")
		}
	})

	t.Run("IsValid_Used", func(t *testing.T) {
		used := time.Now()
		reset.UsedAt = &used
		if reset.IsValid() {
			t.Error("PasswordReset should be invalid when already used")
		}
	})

	t.Run("IsValid_Expired", func(t *testing.T) {
		reset.UsedAt = nil
		reset.ExpiresAt = time.Now().Add(-1 * time.Hour)
		if reset.IsValid() {
			t.Error("PasswordReset should be invalid when expired")
		}
	})

	t.Run("Fields", func(t *testing.T) {
		if reset.ID != "reset123" {
			t.Errorf("ID should be reset123, got %s", reset.ID)
		}
		if reset.UserID != "user123" {
			t.Errorf("UserID should be user123, got %s", reset.UserID)
		}
		if reset.Token != "reset_token_123" {
			t.Errorf("Token should be reset_token_123, got %s", reset.Token)
		}
	})
}

// TestCredentials tests the Credentials struct
func TestCredentials(t *testing.T) {
	creds := &gor.Credentials{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password123",
		MFACode:  "123456",
	}

	t.Run("Fields", func(t *testing.T) {
		if creds.Email != "test@example.com" {
			t.Errorf("Email should be test@example.com, got %s", creds.Email)
		}
		if creds.Username != "testuser" {
			t.Errorf("Username should be testuser, got %s", creds.Username)
		}
		if creds.Password != "password123" {
			t.Errorf("Password should be password123, got %s", creds.Password)
		}
		if creds.MFACode != "123456" {
			t.Errorf("MFACode should be 123456, got %s", creds.MFACode)
		}
	})
}

// TestMFAMethod tests the MFAMethod constants
func TestMFAMethod(t *testing.T) {
	t.Run("Constants", func(t *testing.T) {
		if gor.MFAMethodTOTP != "totp" {
			t.Errorf("MFAMethodTOTP should be 'totp', got %s", gor.MFAMethodTOTP)
		}
		if gor.MFAMethodSMS != "sms" {
			t.Errorf("MFAMethodSMS should be 'sms', got %s", gor.MFAMethodSMS)
		}
		if gor.MFAMethodEmail != "email" {
			t.Errorf("MFAMethodEmail should be 'email', got %s", gor.MFAMethodEmail)
		}
	})
}

// TestMFASetup tests the MFASetup struct
func TestMFASetup(t *testing.T) {
	setup := &gor.MFASetup{
		Secret:      "secret123",
		QRCode:      "qr_code_data",
		BackupCodes: []string{"code1", "code2", "code3"},
	}

	t.Run("Fields", func(t *testing.T) {
		if setup.Secret != "secret123" {
			t.Errorf("Secret should be secret123, got %s", setup.Secret)
		}
		if setup.QRCode != "qr_code_data" {
			t.Errorf("QRCode should be qr_code_data, got %s", setup.QRCode)
		}
		if len(setup.BackupCodes) != 3 {
			t.Errorf("BackupCodes should have 3 codes, got %d", len(setup.BackupCodes))
		}
		if setup.BackupCodes[0] != "code1" {
			t.Errorf("First backup code should be code1, got %s", setup.BackupCodes[0])
		}
	})
}

// TestUserStatus tests the UserStatus constants
func TestUserStatus(t *testing.T) {
	t.Run("Constants", func(t *testing.T) {
		if gor.UserStatusActive != "active" {
			t.Errorf("UserStatusActive should be 'active', got %s", gor.UserStatusActive)
		}
		if gor.UserStatusInactive != "inactive" {
			t.Errorf("UserStatusInactive should be 'inactive', got %s", gor.UserStatusInactive)
		}
		if gor.UserStatusSuspended != "suspended" {
			t.Errorf("UserStatusSuspended should be 'suspended', got %s", gor.UserStatusSuspended)
		}
		if gor.UserStatusBanned != "banned" {
			t.Errorf("UserStatusBanned should be 'banned', got %s", gor.UserStatusBanned)
		}
	})
}

// TestTokenType tests the TokenType constants
func TestTokenType(t *testing.T) {
	t.Run("Constants", func(t *testing.T) {
		if gor.TokenTypeAccess != "access" {
			t.Errorf("TokenTypeAccess should be 'access', got %s", gor.TokenTypeAccess)
		}
		if gor.TokenTypeRefresh != "refresh" {
			t.Errorf("TokenTypeRefresh should be 'refresh', got %s", gor.TokenTypeRefresh)
		}
		if gor.TokenTypeReset != "reset" {
			t.Errorf("TokenTypeReset should be 'reset', got %s", gor.TokenTypeReset)
		}
		if gor.TokenTypeVerify != "verify" {
			t.Errorf("TokenTypeVerify should be 'verify', got %s", gor.TokenTypeVerify)
		}
	})
}

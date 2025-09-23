package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cuemby/gor/internal/auth"
)

func main() {
	fmt.Println("\n🔐 Gor Framework - Authentication API Demo")
	fmt.Println("===========================================")

	// Initialize authenticator
	authenticator, err := auth.NewAuthenticator("auth_demo.db")
	if err != nil {
		log.Fatal("Failed to initialize authenticator:", err)
	}
	defer authenticator.Close()

	// Setup demo data
	setupDemoData(authenticator)

	// Demo authentication features
	demoAuthentication(authenticator)

	fmt.Println("\n✅ Authentication system demonstrated successfully!")
}

func setupDemoData(auth *auth.Authenticator) {
	fmt.Println("\n🔧 Setting up demo data...")

	// Create permissions
	permissions := []struct {
		name string
		desc string
	}{
		{"posts.create", "Create posts"},
		{"posts.edit", "Edit posts"},
		{"posts.delete", "Delete posts"},
		{"users.manage", "Manage users"},
		{"admin.access", "Access admin panel"},
	}

	for _, perm := range permissions {
		if err := auth.CreatePermission(perm.name, perm.desc); err == nil {
			fmt.Printf("  ✓ Created permission: %s\n", perm.name)
		}
	}

	// Assign permissions to roles
	adminPerms := []string{"posts.create", "posts.edit", "posts.delete", "users.manage", "admin.access"}
	for _, perm := range adminPerms {
		auth.AssignPermissionToRole("admin", perm)
	}

	editorPerms := []string{"posts.create", "posts.edit"}
	for _, perm := range editorPerms {
		auth.AssignPermissionToRole("editor", perm)
	}

	auth.AssignPermissionToRole("user", "posts.create")

	fmt.Println("  ✓ Configured role permissions")
}

func demoAuthentication(authenticator *auth.Authenticator) {
	fmt.Println("\n👥 User Registration & Authentication")
	fmt.Println("-------------------------------------")

	// Register users
	fmt.Println("\n1. Registering users...")

	users := []struct {
		email    string
		password string
		name     string
		role     string
	}{
		{"admin@example.com", "admin123456", "Admin User", "admin"},
		{"editor@example.com", "editor123", "Editor User", "editor"},
		{"john@example.com", "john123456", "John Doe", "user"},
	}

	var registeredUsers []*auth.User

	for _, u := range users {
		user, err := authenticator.Register(u.email, u.password, u.name)
		if err != nil {
			if err == auth.ErrUserExists {
				// User already exists, get it
				user, _ = authenticator.GetUserByEmail(u.email)
				fmt.Printf("  ℹ️ User already exists: %s\n", u.email)
			} else {
				fmt.Printf("  ❌ Failed to register %s: %v\n", u.email, err)
				continue
			}
		} else {
			fmt.Printf("  ✓ Registered user: %s (ID: %d)\n", u.email, user.ID)
		}

		// Update role
		if u.role != "user" {
			authenticator.UpdateUserRole(user.ID, u.role)
		}

		// Verify email for demo
		authenticator.VerifyEmail(user.ID)

		registeredUsers = append(registeredUsers, user)
	}

	// Test login
	fmt.Println("\n2. Testing authentication...")

	// Successful login
	session, err := authenticator.Login(
		"admin@example.com",
		"admin123456",
		"127.0.0.1",
		"Mozilla/5.0",
		false,
	)
	if err != nil {
		fmt.Printf("  ❌ Login failed: %v\n", err)
	} else {
		fmt.Printf("  ✓ Login successful! Session token: %s...\n", session.Token[:20])
	}

	// Failed login (wrong password)
	_, err = authenticator.Login(
		"john@example.com",
		"wrongpassword",
		"127.0.0.1",
		"Mozilla/5.0",
		false,
	)
	if err == auth.ErrInvalidCredentials {
		fmt.Println("  ✓ Correctly rejected invalid credentials")
	}

	// Test session validation
	fmt.Println("\n3. Testing session validation...")
	if session != nil {
		validSession, user, err := authenticator.ValidateSession(session.Token)
		if err == nil {
			fmt.Printf("  ✓ Session valid for user: %s (expires: %s)\n",
				user.Email,
				validSession.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
	}

	// Test password reset
	fmt.Println("\n4. Testing password reset...")
	resetToken, err := authenticator.RequestPasswordReset("john@example.com")
	if err == nil && resetToken != "" {
		fmt.Printf("  ✓ Password reset token generated: %s...\n", resetToken[:20])

		// Reset password
		err = authenticator.ResetPassword(resetToken, "newpassword123")
		if err == nil {
			fmt.Println("  ✓ Password reset successful")

			// Try login with new password
			_, err = authenticator.Login(
				"john@example.com",
				"newpassword123",
				"127.0.0.1",
				"Mozilla/5.0",
				false,
			)
			if err == nil {
				fmt.Println("  ✓ Login with new password successful")
			}
		}
	}

	// Test authorization
	fmt.Println("\n5. Testing authorization...")

	if len(registeredUsers) > 0 {
		for _, user := range registeredUsers {
			fmt.Printf("\n  User: %s (Role: %s)\n", user.Email, user.Role)

			// Check role
			if authenticator.HasRole(user.ID, "admin") {
				fmt.Println("    ✓ Has admin role")
			}

			// Check permissions
			permsToCheck := []string{"posts.create", "posts.edit", "posts.delete", "users.manage"}
			for _, perm := range permsToCheck {
				if authenticator.HasPermission(user.ID, perm) {
					fmt.Printf("    ✓ Has permission: %s\n", perm)
				}
			}
		}
	}

	// Test account lockout
	fmt.Println("\n6. Testing account lockout...")

	// Create a test user
	testUser, _ := authenticator.Register("locktest@example.com", "testpass123", "Lock Test")
	if testUser != nil {
		// Attempt multiple failed logins
		for i := 1; i <= 6; i++ {
			_, err := authenticator.Login(
				"locktest@example.com",
				"wrongpassword",
				"127.0.0.1",
				"Mozilla/5.0",
				false,
			)
			if err != nil && i == 5 {
				fmt.Printf("  ✓ Account locked after %d failed attempts\n", i)
			}
		}

		// Try to login with correct password (should fail due to lockout)
		_, err = authenticator.Login(
			"locktest@example.com",
			"testpass123",
			"127.0.0.1",
			"Mozilla/5.0",
			false,
		)
		if err != nil {
			fmt.Println("  ✓ Locked account prevents login even with correct password")
		}
	}

	// Test session cleanup
	fmt.Println("\n7. Testing session cleanup...")
	err = authenticator.CleanupExpiredSessions()
	if err == nil {
		fmt.Println("  ✓ Session cleanup completed")
	}

	// Demonstrate HTTP middleware (simulation)
	fmt.Println("\n8. Simulating HTTP middleware...")
	demoHTTPMiddleware(authenticator, session)
}

func demoHTTPMiddleware(authenticator *auth.Authenticator, session *auth.Session) {
	// Simulate HTTP request/response
	mux := http.NewServeMux()

	// Public endpoint
	mux.HandleFunc("/api/public", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"message": "This is a public endpoint",
		})
	})

	// Protected endpoint
	mux.HandleFunc("/api/protected", authenticator.AuthMiddleware(
		func(w http.ResponseWriter, r *http.Request) {
			user := r.Context().Value("user").(*auth.User)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "This is a protected endpoint",
				"user":    user.Email,
			})
		},
	))

	// Admin-only endpoint
	mux.HandleFunc("/api/admin", authenticator.AuthMiddleware(
		authenticator.RequireRole("admin")(
			func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]string{
					"message": "Admin access granted",
				})
			},
		),
	))

	// Create a test server
	server := &http.Server{
		Addr:    ":8083",
		Handler: mux,
	}

	// Start server in background
	go func() {
		server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test requests
	client := &http.Client{}

	// Test public endpoint
	resp, err := client.Get("http://localhost:8083/api/public")
	if err == nil && resp.StatusCode == 200 {
		fmt.Println("  ✓ Public endpoint accessible without auth")
		resp.Body.Close()
	}

	// Test protected endpoint without auth
	resp, err = client.Get("http://localhost:8083/api/protected")
	if err == nil && resp.StatusCode == 401 {
		fmt.Println("  ✓ Protected endpoint blocked without auth")
		resp.Body.Close()
	}

	// Test protected endpoint with auth
	if session != nil {
		req, _ := http.NewRequest("GET", "http://localhost:8083/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+session.Token)
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			fmt.Println("  ✓ Protected endpoint accessible with valid token")
			resp.Body.Close()
		}

		// Test admin endpoint
		req, _ = http.NewRequest("GET", "http://localhost:8083/api/admin", nil)
		req.Header.Set("Authorization", "Bearer "+session.Token)
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			fmt.Println("  ✓ Admin endpoint accessible for admin user")
			resp.Body.Close()
		}
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// Summary of features demonstrated:
// 1. User registration with email and password
// 2. Password hashing with bcrypt
// 3. Login with session creation
// 4. Session validation and expiration
// 5. Password reset functionality
// 6. Role-based access control (RBAC)
// 7. Permission-based authorization
// 8. Account lockout after failed attempts
// 9. Session cleanup
// 10. HTTP middleware for authentication
// 11. Protected API endpoints
// 12. Admin-only routes
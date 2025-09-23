package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/cuemby/gor/internal/auth"
	"github.com/cuemby/gor/internal/router"
	"github.com/cuemby/gor/pkg/gor"
	"github.com/cuemby/gor/pkg/middleware"
)

var authenticator *auth.Authenticator
var templates *template.Template

type PageData struct {
	Title   string
	User    *auth.User
	Message string
	Error   string
	CSRF    string
}

func init() {
	// Initialize authenticator
	var err error
	authenticator, err = auth.NewAuthenticator("auth_demo.db")
	if err != nil {
		log.Fatal("Failed to initialize authenticator:", err)
	}

	// Setup default admin user and permissions
	setupDefaultData()

	// Parse templates
	templates = template.Must(template.New("").Parse(templateHTML))
}

func setupDefaultData() {
	// Create default permissions
	permissions := []struct {
		name string
		desc string
	}{
		{"users.view", "View users"},
		{"users.edit", "Edit users"},
		{"users.delete", "Delete users"},
		{"admin.access", "Access admin panel"},
	}

	for _, perm := range permissions {
		authenticator.CreatePermission(perm.name, perm.desc)
	}

	// Assign permissions to roles
	authenticator.AssignPermissionToRole("admin", "users.view")
	authenticator.AssignPermissionToRole("admin", "users.edit")
	authenticator.AssignPermissionToRole("admin", "users.delete")
	authenticator.AssignPermissionToRole("admin", "admin.access")

	authenticator.AssignPermissionToRole("user", "users.view")

	// Create admin user if doesn't exist
	adminUser, err := authenticator.Register("admin@example.com", "admin123", "Admin User")
	if err == nil {
		authenticator.UpdateUserRole(adminUser.ID, "admin")
		authenticator.VerifyEmail(adminUser.ID)
		log.Println("Created admin user: admin@example.com / admin123")
	}
}

func main() {
	fmt.Println("\n🔐 Gor Framework - Authentication Demo")
	fmt.Println("=======================================")
	fmt.Println("Visit http://localhost:8082")
	fmt.Println("Default admin: admin@example.com / admin123")
	fmt.Println("Press Ctrl+C to stop\n")

	// Create router
	app := &SimpleApp{}
	appRouter := router.NewRouter(app)

	// Add middleware
	appRouter.Use(
		middleware.Logger(),
		middleware.Recovery(),
	)

	// Public routes
	appRouter.GET("/", handleHome)
	appRouter.GET("/login", handleLoginForm)
	appRouter.POST("/login", handleLogin)
	appRouter.GET("/register", handleRegisterForm)
	appRouter.POST("/register", handleRegister)
	appRouter.GET("/logout", handleLogout)
	appRouter.GET("/forgot-password", handleForgotPasswordForm)
	appRouter.POST("/forgot-password", handleForgotPassword)
	appRouter.GET("/reset-password", handleResetPasswordForm)
	appRouter.POST("/reset-password", handleResetPassword)

	// Protected routes
	appRouter.GET("/dashboard", withAuth(handleDashboard))
	appRouter.GET("/profile", withAuth(handleProfile))
	appRouter.POST("/profile/password", withAuth(handleChangePassword))

	// Admin routes
	appRouter.GET("/admin", withAuth(withRole("admin", handleAdmin)))
	appRouter.GET("/admin/users", withAuth(withPermission("users.view", handleUsers)))

	// Start cleanup routine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			authenticator.CleanupExpiredSessions()
		}
	}()

	// Start server
	server := &http.Server{
		Addr:    ":8082",
		Handler: appRouter,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// Handlers

func handleHome(ctx *gor.Context) error {
	user := getUserFromSession(ctx.Request)
	return renderTemplate(ctx.Response, "home", PageData{
		Title: "Home",
		User:  user,
	})
}

func handleLoginForm(ctx *gor.Context) error {
	return renderTemplate(ctx.Response, "login", PageData{
		Title: "Login",
	})
}

func handleLogin(ctx *gor.Context) error {
	email := ctx.Request.FormValue("email")
	password := ctx.Request.FormValue("password")
	remember := ctx.Request.FormValue("remember") == "on"

	session, err := authenticator.Login(
		email,
		password,
		ctx.Request.RemoteAddr,
		ctx.Request.UserAgent(),
		remember,
	)

	if err != nil {
		return renderTemplate(ctx.Response, "login", PageData{
			Title: "Login",
			Error: err.Error(),
		})
	}

	// Set session cookie
	expiry := time.Now().Add(24 * time.Hour)
	if remember {
		expiry = time.Now().Add(30 * 24 * time.Hour)
	}

	http.SetCookie(ctx.Response, &http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	return ctx.Redirect(http.StatusSeeOther, "/dashboard")
}

func handleRegisterForm(ctx *gor.Context) error {
	return renderTemplate(ctx.Response, "register", PageData{
		Title: "Register",
	})
}

func handleRegister(ctx *gor.Context) error {
	email := ctx.Request.FormValue("email")
	password := ctx.Request.FormValue("password")
	name := ctx.Request.FormValue("name")

	user, err := authenticator.Register(email, password, name)
	if err != nil {
		return renderTemplate(ctx.Response, "register", PageData{
			Title: "Register",
			Error: err.Error(),
		})
	}

	// Auto-login after registration
	session, err := authenticator.CreateSession(
		user.ID,
		ctx.Request.RemoteAddr,
		ctx.Request.UserAgent(),
		24*time.Hour,
	)

	if err == nil {
		http.SetCookie(ctx.Response, &http.Cookie{
			Name:     "session_token",
			Value:    session.Token,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
		})
	}

	return ctx.Redirect(http.StatusSeeOther, "/dashboard")
}

func handleLogout(ctx *gor.Context) error {
	cookie, err := ctx.Request.Cookie("session_token")
	if err == nil {
		authenticator.Logout(cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(ctx.Response, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	return ctx.Redirect(http.StatusSeeOther, "/")
}

func handleDashboard(ctx *gor.Context) error {
	user := getUserFromSession(ctx.Request)
	return renderTemplate(ctx.Response, "dashboard", PageData{
		Title: "Dashboard",
		User:  user,
	})
}

func handleProfile(ctx *gor.Context) error {
	user := getUserFromSession(ctx.Request)
	return renderTemplate(ctx.Response, "profile", PageData{
		Title: "Profile",
		User:  user,
	})
}

func handleChangePassword(ctx *gor.Context) error {
	user := getUserFromSession(ctx.Request)
	if user == nil {
		return ctx.Redirect(http.StatusSeeOther, "/login")
	}

	currentPassword := ctx.Request.FormValue("current_password")
	newPassword := ctx.Request.FormValue("new_password")

	err := authenticator.ChangePassword(user.ID, currentPassword, newPassword)

	message := ""
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	} else {
		message = "Password changed successfully"
	}

	return renderTemplate(ctx.Response, "profile", PageData{
		Title:   "Profile",
		User:    user,
		Message: message,
		Error:   errorMsg,
	})
}

func handleForgotPasswordForm(ctx *gor.Context) error {
	return renderTemplate(ctx.Response, "forgot-password", PageData{
		Title: "Forgot Password",
	})
}

func handleForgotPassword(ctx *gor.Context) error {
	email := ctx.Request.FormValue("email")
	token, _ := authenticator.RequestPasswordReset(email)

	message := "If an account exists with that email, a reset link has been sent."
	if token != "" {
		// In production, send email with reset link
		message = fmt.Sprintf("Reset token (demo only): %s", token)
	}

	return renderTemplate(ctx.Response, "forgot-password", PageData{
		Title:   "Forgot Password",
		Message: message,
	})
}

func handleResetPasswordForm(ctx *gor.Context) error {
	token := ctx.Request.URL.Query().Get("token")
	return renderTemplate(ctx.Response, "reset-password", PageData{
		Title: "Reset Password",
		CSRF:  token,
	})
}

func handleResetPassword(ctx *gor.Context) error {
	token := ctx.Request.FormValue("token")
	newPassword := ctx.Request.FormValue("new_password")

	err := authenticator.ResetPassword(token, newPassword)
	if err != nil {
		return renderTemplate(ctx.Response, "reset-password", PageData{
			Title: "Reset Password",
			Error: err.Error(),
			CSRF:  token,
		})
	}

	return renderTemplate(ctx.Response, "login", PageData{
		Title:   "Login",
		Message: "Password reset successfully. Please login.",
	})
}

func handleAdmin(ctx *gor.Context) error {
	user := getUserFromSession(ctx.Request)
	return renderTemplate(ctx.Response, "admin", PageData{
		Title: "Admin Dashboard",
		User:  user,
	})
}

func handleUsers(ctx *gor.Context) error {
	user := getUserFromSession(ctx.Request)
	return renderTemplate(ctx.Response, "users", PageData{
		Title: "User Management",
		User:  user,
	})
}

// Middleware

func withAuth(handler gor.HandlerFunc) gor.HandlerFunc {
	return func(ctx *gor.Context) error {
		user := getUserFromSession(ctx.Request)
		if user == nil {
			return ctx.Redirect(http.StatusSeeOther, "/login")
		}
		return handler(ctx)
	}
}

func withRole(role string, handler gor.HandlerFunc) gor.HandlerFunc {
	return func(ctx *gor.Context) error {
		user := getUserFromSession(ctx.Request)
		if user == nil || !authenticator.HasRole(user.ID, role) {
			return ctx.HTML(http.StatusForbidden, "<h1>403 - Forbidden</h1>")
		}
		return handler(ctx)
	}
}

func withPermission(permission string, handler gor.HandlerFunc) gor.HandlerFunc {
	return func(ctx *gor.Context) error {
		user := getUserFromSession(ctx.Request)
		if user == nil || !authenticator.HasPermission(user.ID, permission) {
			return ctx.HTML(http.StatusForbidden, "<h1>403 - Forbidden</h1>")
		}
		return handler(ctx)
	}
}

// Helpers

func getUserFromSession(r *http.Request) *auth.User {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil
	}

	_, user, err := authenticator.ValidateSession(cookie.Value)
	if err != nil {
		return nil
	}

	return user
}

func renderTemplate(w http.ResponseWriter, name string, data PageData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return templates.ExecuteTemplate(w, name, data)
}

// Simple App implementation
type SimpleApp struct {
	router gor.Router
}

func (a *SimpleApp) Start(ctx context.Context) error     { return nil }
func (a *SimpleApp) Stop(ctx context.Context) error      { return nil }
func (a *SimpleApp) Router() gor.Router                  { return a.router }
func (a *SimpleApp) ORM() gor.ORM                        { return nil }
func (a *SimpleApp) Queue() gor.Queue                    { return nil }
func (a *SimpleApp) Cache() gor.Cache                    { return nil }
func (a *SimpleApp) Cable() gor.Cable                    { return nil }
func (a *SimpleApp) Auth() interface{}                   { return nil }
func (a *SimpleApp) Config() gor.Config                  { return nil }

// Templates
var templateHTML = `

{{define "home"}}
{{template "layout" .}}
{{define "content"}}
<div class="container">
    <h1>🔐 Gor Authentication Demo</h1>
    {{if .User}}
        <div class="user-info">
            <strong>Welcome back, {{.User.Name}}!</strong><br>
            Email: {{.User.Email}}<br>
            Role: {{.User.Role}}<br>
            Verified: {{if .User.EmailVerified}}✅ Yes{{else}}❌ No{{end}}
        </div>
        <a href="/dashboard"><button>Go to Dashboard</button></a>
    {{else}}
        <p style="margin-bottom: 30px;">Experience Rails-style authentication in Go!</p>
        <a href="/login"><button>Login</button></a>
        <div class="links">
            <a href="/register">Create Account</a> • 
            <a href="/forgot-password">Forgot Password?</a>
        </div>
        <div style="margin-top: 30px; padding: 20px; background: #f8f9fa; border-radius: 5px;">
            <strong>Demo Credentials:</strong><br>
            Email: admin@example.com<br>
            Password: admin123
        </div>
    {{end}}
</div>
{{end}}
{{end}}

{{define "login"}}
{{template "layout" .}}
{{define "content"}}
<div class="container">
    <h1>Login</h1>
    {{if .Message}}<div class="message success">{{.Message}}</div>{{end}}
    {{if .Error}}<div class="message error">{{.Error}}</div>{{end}}
    <form method="POST" action="/login">
        <div class="form-group">
            <label for="email">Email</label>
            <input type="email" id="email" name="email" required>
        </div>
        <div class="form-group">
            <label for="password">Password</label>
            <input type="password" id="password" name="password" required>
        </div>
        <div class="form-group">
            <label>
                <input type="checkbox" name="remember"> Remember me for 30 days
            </label>
        </div>
        <button type="submit">Login</button>
    </form>
    <div class="links">
        <a href="/register">Create Account</a> • 
        <a href="/forgot-password">Forgot Password?</a>
    </div>
</div>
{{end}}
{{end}}

{{define "register"}}
{{template "layout" .}}
{{define "content"}}
<div class="container">
    <h1>Create Account</h1>
    {{if .Error}}<div class="message error">{{.Error}}</div>{{end}}
    <form method="POST" action="/register">
        <div class="form-group">
            <label for="name">Name</label>
            <input type="text" id="name" name="name" required>
        </div>
        <div class="form-group">
            <label for="email">Email</label>
            <input type="email" id="email" name="email" required>
        </div>
        <div class="form-group">
            <label for="password">Password</label>
            <input type="password" id="password" name="password" required minlength="8">
            <small style="color: #666;">Minimum 8 characters</small>
        </div>
        <button type="submit">Create Account</button>
    </form>
    <div class="links">
        <a href="/login">Already have an account? Login</a>
    </div>
</div>
{{end}}
{{end}}

{{define "dashboard"}}
{{template "layout" .}}
{{define "content"}}
<div class="container wide-container">
    <h1>Dashboard</h1>
    <div class="user-info">
        <h2>Welcome, {{.User.Name}}!</h2>
        <p>You are logged in as <strong>{{.User.Email}}</strong></p>
        <p>Role: <strong>{{.User.Role}}</strong></p>
    </div>
    
    <h2>Available Features</h2>
    <ul class="feature-list">
        <li>✅ Secure password hashing with bcrypt</li>
        <li>✅ Session-based authentication</li>
        <li>✅ Remember me functionality</li>
        <li>✅ Password reset capability</li>
        <li>✅ Role-based access control</li>
        <li>✅ Permission system</li>
        <li>✅ Account lockout after failed attempts</li>
        <li>✅ Session expiration and cleanup</li>
    </ul>
    
    {{if eq .User.Role "admin"}}
    <h2>Admin Actions</h2>
    <p><a href="/admin"><button>Go to Admin Panel</button></a></p>
    {{end}}
</div>
{{end}}
{{end}}

{{define "profile"}}
{{template "layout" .}}
{{define "content"}}
<div class="container">
    <h1>Profile</h1>
    {{if .Message}}<div class="message success">{{.Message}}</div>{{end}}
    {{if .Error}}<div class="message error">{{.Error}}</div>{{end}}
    
    <div class="user-info">
        <p><strong>Name:</strong> {{.User.Name}}</p>
        <p><strong>Email:</strong> {{.User.Email}}</p>
        <p><strong>Role:</strong> {{.User.Role}}</p>
        <p><strong>Email Verified:</strong> {{if .User.EmailVerified}}✅ Yes{{else}}❌ No{{end}}</p>
    </div>
    
    <h2>Change Password</h2>
    <form method="POST" action="/profile/password">
        <div class="form-group">
            <label for="current_password">Current Password</label>
            <input type="password" id="current_password" name="current_password" required>
        </div>
        <div class="form-group">
            <label for="new_password">New Password</label>
            <input type="password" id="new_password" name="new_password" required minlength="8">
        </div>
        <button type="submit">Change Password</button>
    </form>
</div>
{{end}}
{{end}}

{{define "forgot-password"}}
{{template "layout" .}}
{{define "content"}}
<div class="container">
    <h1>Forgot Password</h1>
    {{if .Message}}<div class="message success">{{.Message}}</div>{{end}}
    {{if .Error}}<div class="message error">{{.Error}}</div>{{end}}
    <form method="POST" action="/forgot-password">
        <div class="form-group">
            <label for="email">Email</label>
            <input type="email" id="email" name="email" required>
            <small style="color: #666;">Enter your account email to receive a reset link</small>
        </div>
        <button type="submit">Send Reset Link</button>
    </form>
    <div class="links">
        <a href="/login">Back to Login</a>
    </div>
</div>
{{end}}
{{end}}

{{define "reset-password"}}
{{template "layout" .}}
{{define "content"}}
<div class="container">
    <h1>Reset Password</h1>
    {{if .Error}}<div class="message error">{{.Error}}</div>{{end}}
    <form method="POST" action="/reset-password">
        <input type="hidden" name="token" value="{{.CSRF}}">
        <div class="form-group">
            <label for="new_password">New Password</label>
            <input type="password" id="new_password" name="new_password" required minlength="8">
            <small style="color: #666;">Minimum 8 characters</small>
        </div>
        <button type="submit">Reset Password</button>
    </form>
    <div class="links">
        <a href="/login">Back to Login</a>
    </div>
</div>
{{end}}
{{end}}

{{define "admin"}}
{{template "layout" .}}
{{define "content"}}
<div class="container wide-container">
    <h1>Admin Dashboard</h1>
    <div class="user-info">
        <p>Logged in as: <strong>{{.User.Email}}</strong></p>
        <p>Role: <strong>{{.User.Role}}</strong></p>
    </div>
    
    <h2>Admin Functions</h2>
    <ul class="feature-list">
        <li><a href="/admin/users">👥 User Management</a></li>
        <li>📊 Analytics (Coming Soon)</li>
        <li>⚙️ System Settings (Coming Soon)</li>
        <li>📝 Audit Logs (Coming Soon)</li>
    </ul>
</div>
{{end}}
{{end}}

{{define "users"}}
{{template "layout" .}}
{{define "content"}}
<div class="container wide-container">
    <h1>User Management</h1>
    <p>This page demonstrates permission-based access control.</p>
    <p>You have the <strong>users.view</strong> permission to see this page.</p>
    
    <div style="margin-top: 30px; padding: 20px; background: #f8f9fa; border-radius: 5px;">
        <h3>Permission System</h3>
        <ul>
            <li>users.view - View user list</li>
            <li>users.edit - Edit user details</li>
            <li>users.delete - Delete users</li>
            <li>admin.access - Access admin panel</li>
        </ul>
    </div>
</div>
{{end}}
{{end}}
`
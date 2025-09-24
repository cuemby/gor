# Security Policy

## Supported Versions

We actively support security updates for the following versions of Gor:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of Gor seriously. If you discover a security vulnerability, please follow these steps:

### 1. Do NOT create a public issue

Please **do not** report security vulnerabilities through public GitHub issues. This could put users at risk.

### 2. Report privately

Send security reports to: **security@cuemby.com**

Include the following information:
- Description of the vulnerability
- Steps to reproduce the issue
- Affected versions
- Any potential workarounds
- Your assessment of the impact and severity

### 3. What to expect

- **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
- **Initial Assessment**: We will provide an initial assessment within 5 business days
- **Updates**: We will keep you informed of our progress every 7 days until resolution
- **Resolution**: We aim to resolve critical vulnerabilities within 30 days

### 4. Responsible Disclosure

We follow responsible disclosure practices:
- We will work with you to understand and resolve the issue
- We will credit you in our security advisory (unless you prefer to remain anonymous)
- We ask that you do not publicly disclose the vulnerability until we have released a fix

## Security Best Practices

When using Gor in production:

### Database Security
- Use strong, unique passwords for database connections
- Enable SSL/TLS for database connections in production
- Regularly update database software
- Limit database access to necessary users only

### Application Security
- Keep Gor and all dependencies up to date
- Use HTTPS in production environments
- Implement proper input validation and sanitization
- Use secure session management
- Enable CSRF protection (built into Gor)
- Configure CORS appropriately for your use case

### Infrastructure Security
- Run Gor with minimal required privileges
- Use reverse proxy (nginx, Apache) in production
- Implement rate limiting and DDoS protection
- Monitor logs for suspicious activity
- Use security scanning tools in your CI/CD pipeline

### Configuration Security
- Never commit secrets to version control
- Use environment variables for sensitive configuration
- Rotate secrets regularly
- Enable security headers (Gor provides middleware for this)

## Security Features

Gor includes several built-in security features:

### CSRF Protection
```go
// Automatically enabled in production
router.Use(middleware.CSRF())
```

### Security Headers
```go
// Add security headers
router.Use(middleware.Security())
```

### Input Validation
```go
// Built-in validation support
type User struct {
    Email string `validate:"required,email"`
    Name  string `validate:"required,min=2,max=50"`
}
```

### Authentication
- Secure password hashing (bcrypt)
- Session management with secure defaults
- JWT support with proper verification

## Vulnerability Disclosure Timeline

When we receive a security report:

1. **Day 0**: Vulnerability reported
2. **Day 1-2**: Acknowledgment sent to reporter
3. **Day 3-7**: Initial assessment and triage
4. **Day 8-21**: Development of fix and testing
5. **Day 22-28**: Security advisory preparation
6. **Day 29-30**: Release with security fix
7. **Day 31+**: Public disclosure (if appropriate)

## Security Contact

For security-related questions or concerns:
- Email: security@cuemby.com
- For urgent matters, please include "URGENT SECURITY" in the subject line

## Attribution

We appreciate security researchers and will acknowledge contributions in:
- Security advisories
- Release notes
- Our security acknowledgments page (if you consent)

Thank you for helping keep Gor and its users safe!
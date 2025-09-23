package deploy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// DockerBuilder builds and manages Docker images
type DockerBuilder struct {
	appName  string
	registry string
	buildArgs map[string]string
}

// NewDockerBuilder creates a new Docker builder
func NewDockerBuilder(appName, registry string) *DockerBuilder {
	return &DockerBuilder{
		appName:   appName,
		registry:  registry,
		buildArgs: make(map[string]string),
	}
}

// Build builds a Docker image
func (d *DockerBuilder) Build(ctx context.Context, path string, tag string) error {
	// Generate Dockerfile if it doesn't exist
	if err := d.generateDockerfile(path); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Build command
	args := []string{"build"}

	// Add build args
	for key, value := range d.buildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add tag
	image := d.getImageName(tag)
	args = append(args, "-t", image)

	// Add path
	args = append(args, path)

	// Execute docker build
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}

// Push pushes a Docker image to registry
func (d *DockerBuilder) Push(ctx context.Context, tag string) error {
	if d.registry == "" {
		return fmt.Errorf("registry not configured")
	}

	image := d.getImageName(tag)

	// Execute docker push
	cmd := exec.CommandContext(ctx, "docker", "push", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker push failed: %w", err)
	}

	return nil
}

// Tag tags a Docker image
func (d *DockerBuilder) Tag(ctx context.Context, source, target string) error {
	sourceImage := d.getImageName(source)
	targetImage := d.getImageName(target)

	cmd := exec.CommandContext(ctx, "docker", "tag", sourceImage, targetImage)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker tag failed: %w", err)
	}

	return nil
}

// getImageName returns the full image name
func (d *DockerBuilder) getImageName(tag string) string {
	if d.registry != "" {
		return fmt.Sprintf("%s/%s:%s", d.registry, d.appName, tag)
	}
	return fmt.Sprintf("%s:%s", d.appName, tag)
}

// generateDockerfile generates a Dockerfile if it doesn't exist
func (d *DockerBuilder) generateDockerfile(path string) error {
	dockerfilePath := filepath.Join(path, "Dockerfile")

	// Check if Dockerfile already exists
	if _, err := os.Stat(dockerfilePath); err == nil {
		return nil // Dockerfile exists
	}

	// Generate multi-stage Dockerfile
	tmpl := `# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o gor-app ./cmd/app

# Build CLI tool
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gor ./cmd/gor

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata sqlite

# Create app user
RUN addgroup -g 1000 -S app && \
    adduser -u 1000 -S app -G app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gor-app /app/gor-app
COPY --from=builder /app/gor /app/gor

# Copy assets and views if they exist
COPY --from=builder /app/public ./public
COPY --from=builder /app/views ./views
COPY --from=builder /app/config ./config

# Create necessary directories
RUN mkdir -p /app/tmp/pids /app/tmp/cache /app/log && \
    chown -R app:app /app

# Switch to app user
USER app

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["./gor", "health"] || exit 1

# Run the application
CMD ["./gor-app"]
`

	if err := os.WriteFile(dockerfilePath, []byte(tmpl), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Generate .dockerignore if it doesn't exist
	dockerignorePath := filepath.Join(path, ".dockerignore")
	if _, err := os.Stat(dockerignorePath); err != nil {
		ignoreContent := `.git
.gitignore
*.md
.env
.env.*
!.env.example
test/
tmp/
log/
*.log
coverage/
.gor/deploy/
node_modules/
`
		if err := os.WriteFile(dockerignorePath, []byte(ignoreContent), 0644); err != nil {
			return fmt.Errorf("failed to write .dockerignore: %w", err)
		}
	}

	return nil
}

// GenerateDockerCompose generates a docker-compose.yml file
func GenerateDockerCompose(config *Config) error {
	tmpl := `version: '3.8'

services:
  app:
    image: {{.Registry}}/{{.AppName}}:{{.Version}}
    ports:
      - "3000:3000"
    environment:
      {{- range $key, $value := .EnvVars}}
      {{$key}}: {{$value}}
      {{- end}}
      DATABASE_URL: ${DATABASE_URL}
      REDIS_URL: ${REDIS_URL}
      SECRET_KEY_BASE: ${SECRET_KEY_BASE}
    volumes:
      - ./storage:/app/storage
      - ./log:/app/log
    networks:
      - gor-network
    restart: unless-stopped
    {{- if .HealthCheck}}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000{{.HealthCheck.Path}}"]
      interval: {{.HealthCheck.Interval}}
      timeout: {{.HealthCheck.Timeout}}
      retries: {{.HealthCheck.Retries}}
    {{- end}}

  {{- if .Database}}
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: {{.AppName}}_production
      POSTGRES_USER: {{.AppName}}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - gor-network
    restart: unless-stopped
  {{- end}}

  {{- range .Accessories}}
  {{.Name}}:
    image: {{.Image}}
    {{- if .Port}}
    ports:
      - "{{.Port}}:{{.Port}}"
    {{- end}}
    environment:
      {{- range $key, $value := .Env}}
      {{$key}}: {{$value}}
      {{- end}}
    {{- if .Volumes}}
    volumes:
      {{- range .Volumes}}
      - {{.}}
      {{- end}}
    {{- end}}
    networks:
      - gor-network
    restart: unless-stopped
  {{- end}}

  {{- if .SSL.Enabled}}
  traefik:
    image: traefik:v2.10
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik.yml:/etc/traefik/traefik.yml:ro
      - ./acme.json:/acme.json
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(` + "`" + `traefik.{{.AppName}}.local` + "`" + `)"
      - "traefik.http.routers.api.service=api@internal"
    networks:
      - gor-network
    restart: unless-stopped
  {{- end}}

networks:
  gor-network:
    driver: bridge

volumes:
  postgres-data:
  redis-data:
`

	t, err := template.New("docker-compose").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if err := os.WriteFile("docker-compose.yml", buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}

	return nil
}
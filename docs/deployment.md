# Gor Deployment Guide

Complete guide for deploying Gor applications to production.

## Overview

Gor applications compile to a single binary with embedded assets, making deployment straightforward. This guide covers various deployment strategies and best practices.

## Pre-Deployment Checklist

- [ ] All tests passing (`gor test`)
- [ ] Production configuration ready
- [ ] Database migrations prepared
- [ ] Environment variables documented
- [ ] Security review completed
- [ ] Performance benchmarks met
- [ ] Monitoring configured
- [ ] Backup strategy defined
- [ ] Rollback plan ready

## Building for Production

### Basic Build

```bash
# Build with production optimizations
gor build --production

# Or using Go directly
CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-w -s" \
  -o myapp \
  main.go
```

### Build Options

```bash
# Include version information
gor build \
  --ldflags="-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD)"

# Build with embedded assets
gor build --static

# Compress binary with UPX
gor build --compress

# Cross-platform builds
GOOS=linux GOARCH=amd64 gor build -o myapp-linux
GOOS=windows GOARCH=amd64 gor build -o myapp.exe
GOOS=darwin GOARCH=arm64 gor build -o myapp-mac
```

## Environment Configuration

### Environment Variables

```bash
# Required Variables
export GOR_ENV=production
export DATABASE_URL=postgres://user:pass@host/database
export SECRET_KEY_BASE=$(openssl rand -hex 64)

# Optional Variables
export PORT=3000
export BIND_ADDRESS=0.0.0.0
export FORCE_SSL=true
export REDIS_URL=redis://localhost:6379
export SMTP_HOST=smtp.example.com
export SMTP_PORT=587
export SMTP_USER=notifications@example.com
export SMTP_PASS=secret

# Performance Tuning
export GOMAXPROCS=4
export GOR_WORKERS=10
export GOR_POOL_SIZE=25
export GOR_CACHE_SIZE=100MB
```

### Configuration Files

```yaml
# config/production.yml
production:
  server:
    port: ${PORT:3000}
    bind: ${BIND_ADDRESS:0.0.0.0}
    ssl: ${FORCE_SSL:true}
    timeout:
      read: 30s
      write: 30s
      idle: 120s

  database:
    url: ${DATABASE_URL}
    pool: 25
    timeout: 30s
    log: false

  cache:
    driver: redis
    url: ${REDIS_URL}
    ttl: 3600

  queue:
    workers: 10
    retry: 3
    timeout: 300

  security:
    secret_key_base: ${SECRET_KEY_BASE}
    session_timeout: 3600
    csrf_protection: true
    cors_origins: ["https://example.com"]

  logging:
    level: info
    output: stdout
    format: json
```

## Docker Deployment

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=1 GOOS=linux go build \
  -ldflags="-w -s" \
  -o gor \
  main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -s /bin/sh -u 1000 -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gor .
COPY --from=builder /app/config ./config
COPY --from=builder /app/public ./public

# Change ownership
RUN chown -R app:app /app

USER app

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/gor", "health"]

EXPOSE 3000

CMD ["/app/gor", "server"]
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "3000:3000"
    environment:
      - GOR_ENV=production
      - DATABASE_URL=postgres://gor:secret@db:5432/gor_production
      - SECRET_KEY_BASE=${SECRET_KEY_BASE}
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis
    volumes:
      - uploads:/app/public/uploads
      - logs:/app/log
    restart: unless-stopped

  db:
    image: postgres:14
    environment:
      - POSTGRES_USER=gor
      - POSTGRES_PASSWORD=secret
      - POSTGRES_DB=gor_production
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
      - uploads:/app/public/uploads:ro
    depends_on:
      - app
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
  uploads:
  logs:
```

### Docker Commands

```bash
# Build image
docker build -t myapp:latest .

# Run container
docker run -d \
  --name myapp \
  -p 3000:3000 \
  -e GOR_ENV=production \
  -e DATABASE_URL=... \
  myapp:latest

# Using docker-compose
docker-compose up -d
docker-compose logs -f
docker-compose down
```

## Kubernetes Deployment

### Deployment Manifest

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gor-app
  labels:
    app: gor
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gor
  template:
    metadata:
      labels:
        app: gor
    spec:
      containers:
      - name: gor
        image: myregistry.com/gor:latest
        ports:
        - containerPort: 3000
        env:
        - name: GOR_ENV
          value: "production"
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: gor-secrets
              key: database-url
        - name: SECRET_KEY_BASE
          valueFrom:
            secretKeyRef:
              name: gor-secrets
              key: secret-key-base
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 10
```

### Service Manifest

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: gor-service
spec:
  selector:
    app: gor
  ports:
    - protocol: TCP
      port: 80
      targetPort: 3000
  type: LoadBalancer
```

### Ingress Configuration

```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gor-ingress
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - app.example.com
    secretName: gor-tls
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: gor-service
            port:
              number: 80
```

### Kubernetes Commands

```bash
# Create namespace
kubectl create namespace gor-app

# Create secrets
kubectl create secret generic gor-secrets \
  --from-literal=database-url=$DATABASE_URL \
  --from-literal=secret-key-base=$SECRET_KEY_BASE \
  -n gor-app

# Apply configurations
kubectl apply -f k8s/ -n gor-app

# Check deployment status
kubectl get pods -n gor-app
kubectl logs -f deployment/gor-app -n gor-app

# Scale deployment
kubectl scale deployment/gor-app --replicas=5 -n gor-app

# Rolling update
kubectl set image deployment/gor-app gor=myregistry.com/gor:v2 -n gor-app
kubectl rollout status deployment/gor-app -n gor-app

# Rollback if needed
kubectl rollout undo deployment/gor-app -n gor-app
```

## Cloud Platform Deployments

### AWS Elastic Beanstalk

```yaml
# .elasticbeanstalk/config.yml
global:
  application_name: gor-app
  default_platform: Go 1.21
  default_region: us-west-2

deploy:
  artifact: ./gor.zip

# Create deployment package
zip -r gor.zip . -x "*.git*" -x "test/*"

# Deploy
eb init
eb create production
eb deploy
```

### Google Cloud Run

```bash
# Build container
gcloud builds submit --tag gcr.io/PROJECT_ID/gor-app

# Deploy to Cloud Run
gcloud run deploy gor-app \
  --image gcr.io/PROJECT_ID/gor-app \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars="GOR_ENV=production" \
  --set-secrets="DATABASE_URL=database-url:latest"
```

### Heroku

```bash
# Create Procfile
echo "web: ./gor server -p $PORT" > Procfile

# Create app
heroku create myapp

# Set buildpack
heroku buildpacks:set heroku/go

# Configure
heroku config:set GOR_ENV=production
heroku config:set SECRET_KEY_BASE=$(openssl rand -hex 64)

# Deploy
git push heroku main

# Database
heroku addons:create heroku-postgresql:standard-0
heroku run gor db:migrate
```

### DigitalOcean App Platform

```yaml
# .do/app.yaml
name: gor-app
region: nyc
services:
- name: web
  github:
    repo: username/gor-app
    branch: main
  build_command: go build -o gor main.go
  run_command: ./gor server -p 8080
  environment_slug: go
  instance_size_slug: basic-xs
  instance_count: 2
  http_port: 8080
  envs:
  - key: GOR_ENV
    value: production
  - key: DATABASE_URL
    type: SECRET
    value: ${db.DATABASE_URL}

databases:
- name: db
  engine: postgresql
  size: db-s-1vcpu-1gb
```

## Reverse Proxy Configuration

### Nginx

```nginx
# /etc/nginx/sites-available/gor
upstream gor_backend {
    server 127.0.0.1:3000;
    keepalive 64;
}

server {
    listen 80;
    server_name example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name example.com;

    ssl_certificate /etc/ssl/certs/example.com.crt;
    ssl_certificate_key /etc/ssl/private/example.com.key;

    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Static files
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|woff2?)$ {
        root /var/www/gor/public;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    # Application
    location / {
        proxy_pass http://gor_backend;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

### Caddy

```caddyfile
# Caddyfile
example.com {
    reverse_proxy localhost:3000 {
        header_up Host {host}
        header_up X-Real-IP {remote}
        header_up X-Forwarded-For {remote}
        header_up X-Forwarded-Proto {scheme}
    }

    encode gzip

    handle_path /assets/* {
        root * /var/www/gor/public
        file_server
    }
}
```

## Database Management

### Migration Strategy

```bash
# Before deployment
gor db:migrate --dry-run

# During deployment (zero-downtime)
1. Deploy new code (backward compatible)
2. Run migrations
3. Deploy code that uses new schema
4. Clean up old code

# Rollback strategy
gor db:rollback --step 1
```

### Database Backups

```bash
# PostgreSQL backup
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d_%H%M%S).sql

# Automated backups
cat > /etc/cron.d/gor-backup << EOF
0 2 * * * postgres pg_dump $DATABASE_URL | gzip > /backups/gor_\$(date +\%Y\%m\%d).sql.gz
EOF
```

## Monitoring and Health Checks

### Health Check Endpoints

```go
// Implement in your application
func HealthHandler(ctx *gor.Context) error {
    // Basic health check
    return ctx.JSON(200, map[string]string{
        "status": "healthy",
        "version": Version,
    })
}

func ReadinessHandler(ctx *gor.Context) error {
    // Check dependencies
    if err := db.Ping(); err != nil {
        return ctx.JSON(503, map[string]string{
            "status": "not ready",
            "database": "disconnected",
        })
    }

    return ctx.JSON(200, map[string]string{
        "status": "ready",
    })
}
```

### Prometheus Metrics

```go
// Add metrics endpoint
import "github.com/prometheus/client_golang/prometheus/promhttp"

router.GET("/metrics", promhttp.Handler(), "metrics")
```

### Logging

```yaml
# Structured logging configuration
production:
  logging:
    level: info
    format: json
    outputs:
      - stdout
      - file: /var/log/gor/app.log
    fields:
      app: gor
      environment: production
```

## Performance Optimization

### Application Tuning

```bash
# Set GOMAXPROCS
export GOMAXPROCS=$(nproc)

# Enable profiling
gor server --profile

# Memory limits
export GOMEMLIMIT=500MiB
```

### Database Connection Pooling

```yaml
production:
  database:
    pool:
      max: 25
      min: 5
      idle: 10
      lifetime: 3600
```

### Caching Strategy

```yaml
production:
  cache:
    # Use Redis for distributed cache
    driver: redis

    # Page caching
    page_cache: true
    page_cache_ttl: 300

    # Fragment caching
    fragment_cache: true

    # Query caching
    query_cache: true
    query_cache_ttl: 60
```

## Security Best Practices

### SSL/TLS Configuration

```bash
# Generate self-signed certificate (development)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout server.key -out server.crt

# Let's Encrypt (production)
certbot certonly --webroot -w /var/www/gor/public \
  -d example.com -d www.example.com
```

### Security Headers

```go
// Middleware for security headers
func SecurityHeaders(next gor.HandlerFunc) gor.HandlerFunc {
    return func(ctx *gor.Context) error {
        ctx.SetHeader("X-Frame-Options", "SAMEORIGIN")
        ctx.SetHeader("X-Content-Type-Options", "nosniff")
        ctx.SetHeader("X-XSS-Protection", "1; mode=block")
        ctx.SetHeader("Strict-Transport-Security", "max-age=31536000")
        return next(ctx)
    }
}
```

### Environment Security

```bash
# Never commit secrets
echo ".env" >> .gitignore

# Use secret management
# AWS Secrets Manager
aws secretsmanager create-secret --name gor/production

# HashiCorp Vault
vault kv put secret/gor database_url=$DATABASE_URL

# Kubernetes Secrets
kubectl create secret generic gor-secrets --from-env-file=.env
```

## Zero-Downtime Deployment

### Blue-Green Deployment

```bash
# 1. Deploy to green environment
kubectl apply -f k8s/deployment-green.yaml

# 2. Test green environment
curl http://green.example.com/health

# 3. Switch traffic
kubectl patch service gor-service -p '{"spec":{"selector":{"version":"green"}}}'

# 4. Monitor
kubectl logs -f deployment/gor-green

# 5. Remove old version
kubectl delete deployment gor-blue
```

### Rolling Update

```bash
# Update with zero downtime
kubectl set image deployment/gor-app gor=myapp:v2 --record

# Monitor rollout
kubectl rollout status deployment/gor-app

# Rollback if issues
kubectl rollout undo deployment/gor-app
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**
```bash
# Find process using port
lsof -i :3000
# Kill process
kill -9 <PID>
```

2. **Database Connection Failed**
```bash
# Test connection
psql $DATABASE_URL -c "SELECT 1"
# Check firewall rules
```

3. **Memory Issues**
```bash
# Monitor memory
top -p $(pgrep gor)
# Set limits
export GOMEMLIMIT=500MiB
```

4. **SSL Certificate Issues**
```bash
# Verify certificate
openssl s_client -connect example.com:443
# Check expiration
openssl x509 -in server.crt -noout -dates
```

### Debug Mode

```bash
# Enable debug logging
export GOR_ENV=production
export LOG_LEVEL=debug
./gor server

# Enable Go runtime debug
export GODEBUG=gctrace=1
./gor server
```

## Deployment Checklist

### Pre-deployment

- [ ] Run all tests: `gor test`
- [ ] Check code quality: `go vet ./...`
- [ ] Update dependencies: `go mod tidy`
- [ ] Review security: `gosec ./...`
- [ ] Build production binary: `gor build --production`
- [ ] Test in staging environment

### Deployment

- [ ] Backup database
- [ ] Set environment variables
- [ ] Deploy new version
- [ ] Run migrations: `gor db:migrate`
- [ ] Verify health checks
- [ ] Monitor logs

### Post-deployment

- [ ] Verify application functionality
- [ ] Check performance metrics
- [ ] Monitor error rates
- [ ] Update documentation
- [ ] Notify team

## See Also

- [Getting Started](./getting-started.md) - Initial setup
- [CLI Reference](./cli-reference.md) - Build and deploy commands
- [Configuration Guide](./configuration.md) - Environment configuration
- [Testing Guide](./testing-guide.md) - Pre-deployment testing
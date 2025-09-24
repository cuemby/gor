# Gor Framework Documentation Index

This is a comprehensive index of all documentation in the Gor Framework project, organized by category and location.

## 📂 Root Level Documentation

| File | Description | Status |
|------|-------------|--------|
| [README.md](../README.md) | Project overview, quick start, and main documentation | ✅ Current |
| [LICENSE](../LICENSE) | MIT License text | ✅ Current |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Contribution guidelines and development workflow | ✅ Current |
| [CODE_OF_CONDUCT.md](../CODE_OF_CONDUCT.md) | Community standards and behavior guidelines | ✅ Current |
| [SECURITY.md](../SECURITY.md) | Security policy and vulnerability reporting | ✅ Current |
| [SUPPORT.md](../SUPPORT.md) | Getting help and community support | ✅ Current |
| [CHANGELOG.md](../CHANGELOG.md) | Release notes and version history | ✅ Current |
| [AUTHORS.md](../AUTHORS.md) | Contributors and maintainers | ✅ Current |

## 📁 Documentation Directories

### `/docs/` - Main Documentation Hub

| Directory | Purpose | Contents |
|-----------|---------|----------|
| [docs/project/](project/) | Project governance and community | Contributing, governance, code of conduct |
| [docs/security/](security/) | Security policies and guidelines | Security policy, vulnerability management |
| [docs/changelog/](changelog/) | Release documentation | Detailed release notes, migration guides |
| [docs/api/](api/) | API reference documentation | Auto-generated API docs |
| [docs/guides/](guides/) | User guides and tutorials | Step-by-step guides, how-tos |
| [docs/dev/](dev/) | Development documentation | Development tools, architecture docs |

### `/docs/project/` - Project Governance

| File | Description | Last Updated |
|------|-------------|--------------|
| [CONTRIBUTING.md](project/CONTRIBUTING.md) | Detailed contribution guidelines | Auto-synced |
| [CODE_OF_CONDUCT.md](project/CODE_OF_CONDUCT.md) | Community behavior standards | Stable |
| [GOVERNANCE.md](project/GOVERNANCE.md) | Project governance structure | Stable |

### `/docs/security/` - Security Documentation

| File | Description | Last Updated |
|------|-------------|--------------|
| [SECURITY.md](security/SECURITY.md) | Security policy and reporting | Auto-synced |

### `/docs/changelog/` - Release Documentation

| File | Description | Status |
|------|-------------|--------|
| *Future release notes* | Detailed release documentation | Planned |

### `/docs/api/` - API Reference

| File | Description | Generation |
|------|-------------|------------|
| [api-reference.txt](api/api-reference.txt) | Auto-generated Go package docs | `make docs` |
| *Future API docs* | Structured API documentation | Planned |

### `/docs/guides/` - User Guides

| Guide Topic | File | Status |
|-------------|------|--------|
| *Getting Started* | `getting-started.md` | Planned |
| *Installation Guide* | `installation.md` | Planned |
| *First Application* | `first-app.md` | Planned |
| *Testing Guide* | `testing.md` | Planned |
| *Deployment Guide* | `deployment.md` | Planned |

### `/docs/dev/` - Development Documentation

| File | Description | Status |
|------|-------------|--------|
| [CLAUDE.md](dev/CLAUDE.md) | Claude Code development guidance | ✅ Current |
| [llms.txt](dev/llms.txt) | LLM context for development | ✅ Auto-generated |

## 🔧 Build and Development Files

### `/examples/` - Example Applications

| Example | File | Description |
|---------|------|-------------|
| **webapp** | [examples/webapp/](../examples/webapp/) | Full-featured web application |
| **auth_demo** | [examples/auth_demo/](../examples/auth_demo/) | Authentication system demo |
| **solid_trifecta** | [examples/solid_trifecta/](../examples/solid_trifecta/) | Queue, Cache, Cable demo |
| **blog** | [examples/blog/](../examples/blog/) | Simple blog application |
| **testing_demo** | [examples/testing_demo/](../examples/testing_demo/) | Testing framework examples |
| **template_app** | [examples/template_app/](../examples/template_app/) | Template rendering demo |
| **auth_app** | [examples/auth_app/](../examples/auth_app/) | Authentication handlers |
| **realtime_demo** | [examples/realtime_demo/](../examples/realtime_demo/) | Real-time features demo |

### `/scripts/` - Automation Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| [scripts/docs/sync-llms.sh](../scripts/docs/sync-llms.sh) | Auto-generate llms.txt from codebase | `make docs-sync` |
| [scripts/docs/validate-docs.sh](../scripts/docs/validate-docs.sh) | Validate documentation integrity | `make docs-validate` |
| [scripts/docs/update-claude.sh](../scripts/docs/update-claude.sh) | Update CLAUDE.md with current state | Automatic |

### Configuration Files

| File | Purpose | Location |
|------|---------|----------|
| [Makefile](../Makefile) | Build automation and tasks | Root |
| [go.mod](../go.mod) | Go module definition | Root |
| [go.sum](../go.sum) | Go module checksums | Root |
| [.gitignore](../.gitignore) | Git ignore patterns | Root |
| [.github/workflows/ci.yml](../.github/workflows/ci.yml) | GitHub Actions CI/CD | `.github/workflows/` |

## 🏗️ Code Structure Documentation

### Framework Core (`/pkg/gor/`)

| Interface | File | Purpose |
|-----------|------|---------|
| Application | `framework.go` | Main application interface |
| Router | `framework.go` | HTTP routing interface |
| Controller | `framework.go` | Request handler interface |
| Context | `framework.go` | Request/response context |
| ORM | `orm.go` | Database abstraction |
| Auth | `auth.go` | Authentication system |
| Cache | `cache.go` | Caching interface |
| Queue | `queue.go` | Background jobs |
| Cable | `cable.go` | Real-time messaging |

### Implementation (`/internal/`)

| Component | Directory | Purpose |
|-----------|-----------|---------|
| **Router** | `internal/router/` | HTTP routing implementation |
| **ORM** | `internal/orm/` | Database layer with ActiveRecord patterns |
| **Auth** | `internal/auth/` | Authentication and authorization |
| **Cache** | `internal/cache/` | Multi-tier caching system |
| **Queue** | `internal/queue/` | Background job processing |
| **Cable** | `internal/cable/` | WebSocket/SSE real-time features |
| **Views** | `internal/views/` | Template engine and rendering |
| **Assets** | `internal/assets/` | Asset pipeline and processing |
| **CLI** | `internal/cli/` | Command-line interface |
| **Testing** | `internal/testing/` | Testing framework and utilities |
| **Config** | `internal/config/` | Configuration management |
| **Deploy** | `internal/deploy/` | Deployment utilities |
| **Dev** | `internal/dev/` | Development tools |
| **Plugin** | `internal/plugin/` | Plugin system |
| **SSE** | `internal/sse/` | Server-Sent Events |
| **WebSocket** | `internal/websocket/` | WebSocket implementation |

## 📊 Documentation Status

### Completion Status

| Category | Files | Status | Coverage |
|----------|-------|--------|----------|
| **Root Documentation** | 8/8 | ✅ Complete | 100% |
| **Project Governance** | 3/3 | ✅ Complete | 100% |
| **Security Policy** | 1/1 | ✅ Complete | 100% |
| **API Reference** | 1/5 | 🚧 In Progress | 20% |
| **User Guides** | 0/10 | 📋 Planned | 0% |
| **Example Apps** | 8/8 | ✅ Complete | 100% |
| **Build Scripts** | 3/3 | ✅ Complete | 100% |

### Auto-Generated Documentation

| File | Generator | Trigger | Status |
|------|-----------|---------|--------|
| `docs/dev/llms.txt` | `sync-llms.sh` | `make docs-sync` | ✅ Active |
| `docs/api/api-reference.txt` | `go doc` | `make docs` | ✅ Active |
| `docs/dev/CLAUDE.md` | `update-claude.sh` | Coverage updates | ✅ Active |

### Manual Documentation

| Category | Maintenance | Update Frequency |
|----------|-------------|------------------|
| **Root Files** | Manual | As needed |
| **Project Governance** | Manual | Quarterly review |
| **Security Policy** | Manual | Security reviews |
| **User Guides** | Manual | Feature releases |

## 🔄 Documentation Workflows

### Update Workflows

```bash
# Full documentation update
make docs-all

# Individual operations
make docs-sync          # Update auto-generated docs
make docs-validate      # Check documentation integrity
make docs               # Generate API reference
make docs-update-claude # Update development docs
```

### Validation Workflows

```bash
# Validate documentation
./scripts/docs/validate-docs.sh

# Check file references
grep -r "docs/" . --include="*.md"

# Verify examples compile
go build ./examples/...
```

## 📝 Contributing to Documentation

### Adding New Documentation

1. **Choose the right location** based on audience and purpose
2. **Follow naming conventions** (kebab-case for files)
3. **Update this index** when adding new files
4. **Add cross-references** to related documentation
5. **Test all code examples** before submitting
6. **Run validation** with `make docs-validate`

### Documentation Standards

- **Markdown format** for all documentation files
- **Clear headings** with logical hierarchy
- **Code examples** that actually work
- **Links verification** with validation scripts
- **Consistent formatting** following project style

---

**Documentation Index Last Updated**: Auto-generated • **Total Files Documented**: 50+

*This index is maintained manually but validated automatically. Run `make docs-validate` to check integrity.*
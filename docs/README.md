# Gor Framework Documentation

Welcome to the comprehensive documentation for the Gor Framework - a Rails-inspired web framework for Go.

## 📚 Documentation Structure

This documentation is organized into logical sections to help you find what you need quickly:

### 📖 For Users

- **[Getting Started](../README.md)** - Quick start guide and project overview
- **[API Documentation](api/)** - Complete API reference (generated from code)
- **[User Guides](guides/)** - Step-by-step tutorials and how-to guides
- **[Examples](../examples/)** - Working example applications

### 🛠️ For Contributors

- **[Contributing](../CONTRIBUTING.md)** - How to contribute to Gor
- **[Development Documentation](dev/)** - Development tools and guidelines
- **[Project Governance](project/)** - Community guidelines and policies

### 🔒 For Security

- **[Security Policy](../SECURITY.md)** - Vulnerability reporting and security guidelines
- **[Security Documentation](security/)** - Security policies and best practices

### 📝 For Maintainers

- **[Changelog](../CHANGELOG.md)** - Release notes and version history
- **[Release Documentation](changelog/)** - Detailed release information

## 🗺️ Quick Navigation

### Most Common Documentation

| What you want to do | Where to look |
|-------------------|---------------|
| **Get started with Gor** | [README.md](../README.md) |
| **Learn framework concepts** | [User Guides](guides/) |
| **Find API reference** | [API Documentation](api/) |
| **See working examples** | [Examples](../examples/) |
| **Contribute code** | [CONTRIBUTING.md](../CONTRIBUTING.md) |
| **Report security issues** | [SECURITY.md](../SECURITY.md) |
| **Get community support** | [SUPPORT.md](../SUPPORT.md) |

### By User Type

**🚀 New Users**: Start with [README.md](../README.md) → [Examples](../examples/) → [User Guides](guides/)

**📖 Developers**: [API Documentation](api/) → [Examples](../examples/) → [Development Docs](dev/)

**🤝 Contributors**: [CONTRIBUTING.md](../CONTRIBUTING.md) → [Development Docs](dev/) → [Project Governance](project/)

**🔧 Maintainers**: [Development Docs](dev/) → [Release Docs](changelog/) → [Project Governance](project/)

## 🏗️ Framework Architecture

Gor follows a clear architectural pattern:

- **`pkg/gor/`** - Core framework interfaces and contracts
- **`internal/`** - Implementation of framework components
- **`cmd/gor/`** - CLI application and tools
- **`examples/`** - Demonstration applications

### Key Features

- ✅ **Convention Over Configuration** - Rails-like productivity
- ✅ **No Redis Required** - Database-backed queue, cache, and real-time features
- ✅ **Type Safety** - Compile-time checking with Go's type system
- ✅ **Single Binary** - Deploy one file with embedded assets
- ✅ **The Solid Trifecta** - Queue, Cache, and Cable systems included

## 🔄 Documentation Automation

This documentation is automatically kept in sync with the codebase using:

- **`scripts/docs/sync-llms.sh`** - Updates documentation from code analysis
- **`scripts/docs/validate-docs.sh`** - Validates documentation integrity
- **`scripts/docs/update-claude.sh`** - Keeps development docs current

To update documentation:
```bash
# Sync all documentation
make docs-sync

# Validate documentation
make docs-validate

# Generate API docs
make docs

# Full documentation workflow
make docs-all
```

## 📋 Documentation Standards

### Writing Guidelines

- **Clear and Concise**: Write for your audience level
- **Examples First**: Show working code before explaining concepts
- **Keep Current**: Documentation is automatically validated
- **Cross-Reference**: Link to related documentation
- **Test Code**: All code examples should work

### File Organization

- **Logical Grouping**: Related docs in same directory
- **Clear Naming**: Descriptive file names
- **Index Files**: README.md in each directory
- **Consistent Format**: Follow established patterns

## 🤔 Getting Help

- **General Questions**: [GitHub Discussions](https://github.com/cuemby/gor/discussions)
- **Bug Reports**: [GitHub Issues](https://github.com/cuemby/gor/issues)
- **Security Issues**: See [SECURITY.md](../SECURITY.md)
- **Contributing Help**: See [CONTRIBUTING.md](../CONTRIBUTING.md)

## 📊 Project Status

- **Version**: 1.0.0 (Production Ready)
- **Test Coverage**: ~75% and improving
- **License**: MIT
- **Go Version**: 1.21+

---

## 📖 Detailed Documentation Index

For a comprehensive list of all documentation files, see [INDEX.md](INDEX.md).

**Happy coding with Gor!** 🚀
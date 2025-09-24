# Project Structure Optimization - Migration Checklist

## Current State Documentation (Pre-Migration)

### Root Directory Files (15 files)
```
├── CHANGELOG.md              # → docs/changelog/
├── CLAUDE.md                 # → docs/dev/ (development documentation)
├── CODE_OF_CONDUCT.md        # → docs/project/
├── CONTRIBUTING.md           # → docs/project/
├── GOVERNANCE.md             # → docs/project/
├── LICENSE                   # → KEEP in root (legal requirement)
├── Makefile                  # → KEEP in root (build automation)
├── README.md                 # → KEEP in root (GitHub requirement)
├── SECURITY.md               # → docs/security/
├── go.mod                    # → KEEP in root (Go requirement)
├── go.sum                    # → KEEP in root (Go requirement)
├── llms.txt                  # → docs/dev/ (LLM documentation)
├── .gitignore                # → KEEP in root (optimize content)
├── .gosec.toml               # → build/security/
└── gosec.sarif               # → tmp/ or remove (build artifact)
```

### Current Directory Structure
```
/
├── [root files above]
├── .github/
│   └── workflows/
│       └── ci.yml
├── cmd/
│   └── gor/
├── examples/
│   ├── auth_app/
│   ├── auth_demo/
│   ├── blog/
│   ├── realtime_demo/
│   ├── solid_trifecta/
│   ├── template_app/
│   ├── testing_demo/
│   ├── webapp/
│   └── orm_example.go
├── internal/
│   ├── app/
│   ├── assets/
│   ├── auth/
│   ├── cable/
│   ├── cache/
│   ├── cli/
│   ├── config/
│   ├── deploy/
│   ├── dev/
│   ├── orm/
│   ├── plugin/
│   ├── queue/
│   ├── router/
│   ├── sse/
│   ├── testing/
│   ├── views/
│   └── websocket/
└── pkg/
    ├── gor/
    └── middleware/
```

## Target Structure (Post-Migration)

### Clean Root Directory (≤8 files)
```
/
├── README.md                 # Project overview (essential)
├── LICENSE                   # Legal requirement
├── Makefile                  # Build automation
├── go.mod                    # Go dependencies
├── go.sum                    # Go dependencies
├── .gitignore                # Git configuration (optimized)
└── [directories only]       # No other loose files
```

### New Organized Structure
```
/
├── [clean root files above]
├── cmd/                      # CLI applications (unchanged)
├── pkg/                      # Public API (unchanged)
├── internal/                 # Private implementation (unchanged)
├── examples/                 # Example applications (unchanged)
├── docs/                     # All documentation (NEW)
│   ├── project/             # CONTRIBUTING.md, CODE_OF_CONDUCT.md, GOVERNANCE.md
│   ├── security/            # SECURITY.md
│   ├── changelog/           # CHANGELOG.md
│   ├── api/                 # API documentation (generated)
│   ├── guides/              # User guides (future)
│   └── dev/                 # CLAUDE.md, llms.txt, development docs
├── scripts/                 # Automation scripts (NEW)
│   ├── docs/                # Documentation automation
│   ├── build/               # Build scripts
│   └── dev/                 # Development utilities
├── build/                   # Build configurations (NEW)
│   └── security/            # .gosec.toml
├── .github/                 # GitHub configuration (existing)
└── tmp/                     # Temporary files (NEW, gitignored)
```

## Migration Tasks Checklist

### Phase 1: Structure Reorganization
- [ ] Create new directories: docs/, scripts/, build/, tmp/
- [ ] Move CONTRIBUTING.md → docs/project/
- [ ] Move CODE_OF_CONDUCT.md → docs/project/
- [ ] Move GOVERNANCE.md → docs/project/
- [ ] Move SECURITY.md → docs/security/
- [ ] Move CHANGELOG.md → docs/changelog/
- [ ] Move CLAUDE.md → docs/dev/
- [ ] Move llms.txt → docs/dev/
- [ ] Move .gosec.toml → build/security/
- [ ] Remove/ignore gosec.sarif

### Phase 2: .gitignore Optimization
- [ ] Backup current .gitignore → docs/dev/gitignore-backup.old
- [ ] Remove duplicate entries (.DS_Store, Thumbs.db)
- [ ] Fix broken entries (lines 271-273)
- [ ] Reorganize by category with headers
- [ ] Add modern IDE patterns (Cursor, Claude Code)
- [ ] Add tmp/ directory ignore
- [ ] Test new .gitignore works correctly

### Phase 3: Documentation Automation
- [ ] Create scripts/docs/sync-llms.sh
- [ ] Create scripts/docs/validate-docs.sh
- [ ] Create scripts/docs/update-claude.sh
- [ ] Add Makefile targets: docs-sync, docs-validate
- [ ] Test documentation automation workflow

### Phase 4: Update References
- [ ] Update Makefile paths if needed
- [ ] Update GitHub Actions workflow paths
- [ ] Update any hardcoded file references
- [ ] Update import paths if affected
- [ ] Update example documentation

### Phase 5: Validation
- [ ] Test build process: `make build`
- [ ] Run all tests: `make test`
- [ ] Run examples: `make run-webapp`, etc.
- [ ] Validate CI/CD pipeline
- [ ] Check git status is clean
- [ ] Verify documentation automation works

## Risk Assessment & Mitigation

### High Risk Areas
1. **Makefile paths** - May need updates for moved files
2. **GitHub Actions** - May reference moved configuration files
3. **Documentation links** - Internal references may break
4. **Import paths** - Should be unaffected but need verification

### Mitigation Strategies
- Use `git mv` to preserve file history
- Create commits for each logical group of changes
- Test functionality after each major phase
- Keep backup branch for quick rollback

### Rollback Strategy
1. Switch to backup branch: `git checkout backup/pre-structure-optimization`
2. Create new main from backup: `git checkout -b main-rollback backup/pre-structure-optimization`
3. Force push if necessary (with team approval)

## Success Criteria

### Structure Goals
- [ ] Root directory has ≤8 files (currently 15+)
- [ ] All documentation organized in /docs/ hierarchy
- [ ] Configuration files properly categorized
- [ ] Clean separation of concerns

### Functionality Goals
- [ ] All existing functionality preserved
- [ ] Build process works identically
- [ ] All tests pass
- [ ] All examples run correctly
- [ ] CI/CD pipeline functions normally

### Documentation Goals
- [ ] Automated llms.txt generation working
- [ ] CLAUDE.md stays current automatically
- [ ] All file references are valid
- [ ] Documentation validation integrated

## Notes & Decisions

- Keep README.md, LICENSE, Makefile in root (essential files)
- Use git mv to preserve file history
- Test at each phase to catch issues early
- Create comprehensive PR with before/after comparison
- Get team review before merging to main

---
Created: 2025-09-24
Branch: feature/project-structure-optimization
Status: In Progress
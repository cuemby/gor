## Description

<!-- Provide a brief description of the changes in this PR -->

## Type of Change

<!-- Mark the relevant option with an "x" -->

- [ ] 🐛 Bug fix (non-breaking change which fixes an issue)
- [ ] ✨ New feature (non-breaking change which adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📚 Documentation update (changes to documentation only)
- [ ] 🧹 Code refactoring (no functional changes, code improvements)
- [ ] ⚡ Performance improvement
- [ ] 🧪 Test improvements or additions
- [ ] 🔧 Infrastructure/tooling changes

## Related Issues

<!-- Link to related issues using "Fixes #123", "Closes #123", or "Related to #123" -->

- Fixes #
- Related to #

## Changes Made

<!-- Provide a detailed description of the changes -->

### Core Changes
-
-

### API Changes (if applicable)
-
-

### Database Changes (if applicable)
-
-

## Testing

<!-- Describe the testing you've done -->

### Test Coverage
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed
- [ ] All existing tests pass

### Test Commands
```bash
# Commands used to test the changes
make test
make test-coverage
```

### Test Results
<!-- Include any relevant test output or coverage information -->

## Performance Impact

<!-- If applicable, describe any performance implications -->

- [ ] No performance impact
- [ ] Performance improved
- [ ] Performance regression (explain why acceptable)
- [ ] Benchmarks added/updated

### Benchmark Results (if applicable)
```
<!-- Include benchmark results here -->
```

## Documentation

- [ ] Documentation updated (if needed)
- [ ] Code comments added/updated
- [ ] Examples updated (if needed)
- [ ] CHANGELOG.md updated

## Breaking Changes

<!-- If this is a breaking change, describe what breaks and provide migration guide -->

### What breaks:
-

### Migration guide:
```go
// Before
old_code()

// After
new_code()
```

## Security Considerations

- [ ] No security impact
- [ ] Security improvement
- [ ] Potential security implications (explained below)

<!-- If there are security implications, explain them -->

## Deployment Notes

<!-- Any special considerations for deployment -->

- [ ] No special deployment requirements
- [ ] Database migration required
- [ ] Configuration changes required
- [ ] Infrastructure changes required

## Checklist

<!-- Ensure all items are checked before requesting review -->

### Code Quality
- [ ] Code follows the project's style guidelines
- [ ] Self-review of code completed
- [ ] Code is self-documenting with appropriate comments
- [ ] No TODO comments left in the code (unless tracked in issues)

### Testing
- [ ] Tests added for new functionality
- [ ] All tests pass locally
- [ ] Test coverage maintained or improved
- [ ] Edge cases considered and tested

### Documentation
- [ ] Documentation updated for user-facing changes
- [ ] API documentation updated (if applicable)
- [ ] Examples updated (if applicable)
- [ ] CHANGELOG.md entry added

### Compatibility
- [ ] Changes are backwards compatible
- [ ] If breaking changes exist, migration guide provided
- [ ] Go version compatibility maintained (1.21+)
- [ ] Cross-platform compatibility verified

## Screenshots (if applicable)

<!-- Add screenshots for UI changes -->

## Additional Notes

<!-- Any additional information reviewers should know -->

## Review Guidelines

<!-- For reviewers -->

### Focus Areas
Please pay special attention to:
- [ ] Code correctness and edge cases
- [ ] Performance implications
- [ ] Security considerations
- [ ] API design consistency
- [ ] Test coverage adequacy
- [ ] Documentation completeness

### Testing Suggestions
```bash
# Suggested testing commands for reviewers
make ci
make test-coverage
make run-examples
```

---

**For Maintainers:**
- [ ] Release notes impact: major/minor/patch
- [ ] Needs documentation team review
- [ ] Needs security team review
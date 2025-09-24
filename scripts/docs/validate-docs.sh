#!/bin/bash

# validate-docs.sh - Validate documentation consistency
# This script checks for broken references, missing files, and inconsistencies

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

echo -e "${BLUE}🔍 Validating documentation consistency...${NC}"

# Helper function to increment counters
check_result() {
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    if [ $1 -eq 0 ]; then
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        echo -e "${GREEN}✅ $2${NC}"
    else
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        echo -e "${RED}❌ $2${NC}"
    fi
}

# Check 1: Validate all file references in documentation exist
echo -e "\n${YELLOW}📁 Checking file references...${NC}"

# Function to check file references in a document
check_file_references() {
    local doc_file="$1"
    local errors=0

    if [ ! -f "$doc_file" ]; then
        return 1
    fi

    # Extract file references like ./path/to/file.md, ./internal/package, etc.
    grep -oE '\./[^)]*' "$doc_file" 2>/dev/null | while read -r ref; do
        # Clean up the reference (remove trailing punctuation)
        clean_ref=$(echo "$ref" | sed 's/[,;.:)]$//')
        full_path="$PROJECT_ROOT/$clean_ref"

        if [ ! -e "$full_path" ]; then
            echo -e "${RED}  Missing: $clean_ref (referenced in $(basename "$doc_file"))${NC}"
            errors=$((errors + 1))
        fi
    done

    return $errors
}

# Check documentation files
docs_errors=0
for doc in "$PROJECT_ROOT/README.md" \
          "$PROJECT_ROOT/docs/dev/llms.txt" \
          "$PROJECT_ROOT/docs/dev/CLAUDE.md" \
          "$PROJECT_ROOT/docs/project/CONTRIBUTING.md" \
          "$PROJECT_ROOT/CONTRIBUTING.md" \
          "$PROJECT_ROOT/CODE_OF_CONDUCT.md" \
          "$PROJECT_ROOT/SECURITY.md" \
          "$PROJECT_ROOT/SUPPORT.md" \
          "$PROJECT_ROOT/CHANGELOG.md" \
          "$PROJECT_ROOT/AUTHORS.md"; do
    if [ -f "$doc" ]; then
        if ! check_file_references "$doc"; then
            docs_errors=$((docs_errors + 1))
        fi
    fi
done

check_result $docs_errors "File references validation"

# Check 2: Validate example applications compile
echo -e "\n${YELLOW}⚙️  Checking example applications compile...${NC}"

compile_errors=0
if [ -d "$PROJECT_ROOT/examples" ]; then
    for example_dir in "$PROJECT_ROOT/examples"/*/; do
        if [ -f "$example_dir/main.go" ]; then
            example_name=$(basename "$example_dir")
            if ! go build -o /tmp/test_build "$example_dir" >/dev/null 2>&1; then
                echo -e "${RED}  Failed to compile: $example_name${NC}"
                compile_errors=$((compile_errors + 1))
            else
                echo -e "${GREEN}  ✓ $example_name compiles${NC}"
                rm -f /tmp/test_build
            fi
        fi
    done
else
    compile_errors=1
    echo -e "${RED}  Examples directory not found${NC}"
fi

check_result $compile_errors "Example applications compilation"

# Check 3: Validate Go package documentation exists
echo -e "\n${YELLOW}📦 Checking Go package documentation...${NC}"

package_doc_errors=0
for pkg_dir in "$PROJECT_ROOT/pkg"/*/ "$PROJECT_ROOT/internal"/*/; do
    if [ -d "$pkg_dir" ]; then
        pkg_name=$(basename "$pkg_dir")
        main_file="$pkg_dir/$pkg_name.go"

        # Check if main package file exists and has package comment
        if [ -f "$main_file" ]; then
            if ! grep -q "^// Package $pkg_name" "$main_file"; then
                echo -e "${YELLOW}  Warning: $pkg_name missing package documentation${NC}"
                # Don't count as error, just warning
            else
                echo -e "${GREEN}  ✓ $pkg_name documented${NC}"
            fi
        fi
    fi
done

check_result $package_doc_errors "Go package documentation"

# Check 4: Validate project structure consistency
echo -e "\n${YELLOW}🏗️  Checking project structure...${NC}"

structure_errors=0

# Check required directories exist
required_dirs=("cmd" "pkg" "internal" "examples" "docs" "scripts")
for dir in "${required_dirs[@]}"; do
    if [ ! -d "$PROJECT_ROOT/$dir" ]; then
        echo -e "${RED}  Missing required directory: $dir${NC}"
        structure_errors=$((structure_errors + 1))
    else
        echo -e "${GREEN}  ✓ $dir directory exists${NC}"
    fi
done

# Check documentation structure
doc_dirs=("docs/project" "docs/security" "docs/changelog" "docs/dev")
for dir in "${doc_dirs[@]}"; do
    if [ ! -d "$PROJECT_ROOT/$dir" ]; then
        echo -e "${RED}  Missing documentation directory: $dir${NC}"
        structure_errors=$((structure_errors + 1))
    else
        echo -e "${GREEN}  ✓ $dir exists${NC}"
    fi
done

# Check community health files in root directory for GitHub visibility
community_files=("README.md" "LICENSE" "CONTRIBUTING.md" "CODE_OF_CONDUCT.md" "SECURITY.md" "SUPPORT.md" "CHANGELOG.md" "AUTHORS.md")
for file in "${community_files[@]}"; do
    if [ ! -f "$PROJECT_ROOT/$file" ]; then
        echo -e "${RED}  Missing community health file: $file${NC}"
        structure_errors=$((structure_errors + 1))
    else
        echo -e "${GREEN}  ✓ $file exists in root${NC}"
    fi
done

check_result $structure_errors "Project structure consistency"

# Check 5: Validate Makefile targets
echo -e "\n${YELLOW}🔧 Checking Makefile targets...${NC}"

makefile_errors=0
if [ -f "$PROJECT_ROOT/Makefile" ]; then
    # Check essential targets exist
    essential_targets=("build" "test" "clean" "help")
    for target in "${essential_targets[@]}"; do
        if ! grep -q "^$target:" "$PROJECT_ROOT/Makefile"; then
            echo -e "${RED}  Missing Makefile target: $target${NC}"
            makefile_errors=$((makefile_errors + 1))
        else
            echo -e "${GREEN}  ✓ $target target exists${NC}"
        fi
    done
else
    echo -e "${RED}  Makefile not found${NC}"
    makefile_errors=1
fi

check_result $makefile_errors "Makefile targets validation"

# Check 6: Validate Git configuration
echo -e "\n${YELLOW}🔄 Checking Git configuration...${NC}"

git_errors=0

# Check .gitignore exists and has essential patterns
if [ -f "$PROJECT_ROOT/.gitignore" ]; then
    essential_patterns=("*.exe" "*.dll" "*.so" ".DS_Store" "tmp/" "*.log")
    for pattern in "${essential_patterns[@]}"; do
        if ! grep -q "$pattern" "$PROJECT_ROOT/.gitignore"; then
            echo -e "${YELLOW}  Warning: .gitignore missing pattern: $pattern${NC}"
        fi
    done
    echo -e "${GREEN}  ✓ .gitignore exists${NC}"
else
    echo -e "${RED}  .gitignore not found${NC}"
    git_errors=1
fi

# Check GitHub Actions workflow exists
if [ -f "$PROJECT_ROOT/.github/workflows/ci.yml" ]; then
    echo -e "${GREEN}  ✓ GitHub Actions CI workflow exists${NC}"
else
    echo -e "${RED}  Missing GitHub Actions CI workflow${NC}"
    git_errors=1
fi

check_result $git_errors "Git configuration validation"

# Check 7: Validate test coverage can be generated
echo -e "\n${YELLOW}🧪 Checking test infrastructure...${NC}"

test_errors=0

# Check if tests can run
if go test ./... >/dev/null 2>&1; then
    echo -e "${GREEN}  ✓ Tests run successfully${NC}"
else
    echo -e "${RED}  Tests failing${NC}"
    test_errors=1
fi

# Check if coverage can be generated
if go test -coverprofile=/tmp/coverage.out ./... >/dev/null 2>&1; then
    echo -e "${GREEN}  ✓ Test coverage can be generated${NC}"
    rm -f /tmp/coverage.out
else
    echo -e "${YELLOW}  Warning: Test coverage generation failed${NC}"
    # Don't count as error since tests might have issues
fi

check_result $test_errors "Test infrastructure validation"

# Summary
echo -e "\n${BLUE}📊 Validation Summary${NC}"
echo -e "Total checks: $TOTAL_CHECKS"
echo -e "${GREEN}Passed: $PASSED_CHECKS${NC}"
echo -e "${RED}Failed: $FAILED_CHECKS${NC}"

if [ $FAILED_CHECKS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All documentation validation checks passed!${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  $FAILED_CHECKS validation check(s) failed.${NC}"
    echo -e "${YELLOW}Please address the issues above and run validation again.${NC}"
    exit 1
fi
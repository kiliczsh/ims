#!/bin/bash

# IMS Release Management Script
# Usage: ./scripts/release.sh [major|minor|patch|<version>]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Files
VERSION_FILE="$PROJECT_ROOT/VERSION"
CHANGELOG_FILE="$PROJECT_ROOT/CHANGELOG.md"

# Functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if we're in a git repository
check_git_repo() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "This is not a git repository"
        exit 1
    fi
}

# Check if working directory is clean
check_working_directory() {
    if ! git diff-index --quiet HEAD --; then
        log_error "Working directory is not clean. Please commit or stash your changes."
        exit 1
    fi
}

# Get current version
get_current_version() {
    if [ -f "$VERSION_FILE" ]; then
        cat "$VERSION_FILE"
    else
        echo "0.0.0"
    fi
}

# Parse version into components
parse_version() {
    local version=$1
    echo "$version" | sed -E 's/^v?([0-9]+)\.([0-9]+)\.([0-9]+).*$/\1 \2 \3/'
}

# Increment version
increment_version() {
    local current_version=$1
    local increment_type=$2
    
    read -r major minor patch <<< "$(parse_version "$current_version")"
    
    case "$increment_type" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            log_error "Invalid increment type: $increment_type"
            exit 1
            ;;
    esac
    
    echo "$major.$minor.$patch"
}

# Validate version format
validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        log_error "Invalid version format: $version (expected: major.minor.patch)"
        exit 1
    fi
}

# Update version file
update_version_file() {
    local new_version=$1
    echo "$new_version" > "$VERSION_FILE"
    log_success "Updated VERSION file to $new_version"
}

# Update go.mod version (if needed)
update_go_mod() {
    local new_version=$1
    if [ -f "$PROJECT_ROOT/go.mod" ]; then
        # Add version comment to go.mod if not present
        if ! grep -q "// version:" "$PROJECT_ROOT/go.mod"; then
            sed -i.bak '1a\
// version: '"$new_version"'' "$PROJECT_ROOT/go.mod"
            rm -f "$PROJECT_ROOT/go.mod.bak"
            log_success "Updated go.mod with version $new_version"
        else
            sed -i.bak 's|// version:.*|// version: '"$new_version"'|' "$PROJECT_ROOT/go.mod"
            rm -f "$PROJECT_ROOT/go.mod.bak"
            log_success "Updated go.mod version to $new_version"
        fi
    fi
}

# Create or update changelog
update_changelog() {
    local new_version=$1
    local date=$(date +"%Y-%m-%d")
    
    if [ ! -f "$CHANGELOG_FILE" ]; then
        cat > "$CHANGELOG_FILE" << EOF
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [$new_version] - $date

### Added
- Initial release
- Message scheduling and webhook delivery
- Comprehensive audit logging
- Interactive API documentation with Swagger
- Docker support with multi-stage builds
- Health monitoring and status endpoints
- Batch processing with configurable intervals

### Security
- Non-root container execution
- API key authentication
- Input validation and sanitization
EOF
        log_success "Created CHANGELOG.md"
    else
        # Insert new version after [Unreleased] section
        sed -i.bak "/## \[Unreleased\]/a\\
\\
## [$new_version] - $date\\
\\
### Added\\
- \\
\\
### Changed\\
- \\
\\
### Fixed\\
- \\
" "$CHANGELOG_FILE"
        rm -f "$CHANGELOG_FILE.bak"
        log_success "Updated CHANGELOG.md with version $new_version"
    fi
}

# Build release binaries
build_release_binaries() {
    local version=$1
    local build_dir="$PROJECT_ROOT/dist/v$version"
    
    log_info "Building release binaries for version $version..."
    
    # Create build directory
    mkdir -p "$build_dir"
    
    # Build for different platforms
    platforms=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )
    
    for platform in "${platforms[@]}"; do
        IFS='/' read -r os arch <<< "$platform"
        output_name="ims-v${version}-${os}-${arch}"
        
        if [ "$os" = "windows" ]; then
            output_name="${output_name}.exe"
        fi
        
        log_info "Building for $os/$arch..."
        
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
            -ldflags "-X main.version=$version -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.gitCommit=$(git rev-parse --short HEAD)" \
            -o "$build_dir/$output_name" \
            "$PROJECT_ROOT/cmd/server/main.go"
        
        # Create checksum
        if command -v sha256sum >/dev/null 2>&1; then
            (cd "$build_dir" && sha256sum "$output_name" > "${output_name}.sha256")
        elif command -v shasum >/dev/null 2>&1; then
            (cd "$build_dir" && shasum -a 256 "$output_name" > "${output_name}.sha256")
        fi
        
        log_success "Built $output_name"
    done
    
    # Create archive
    cd "$PROJECT_ROOT/dist"
    tar -czf "ims-v${version}.tar.gz" "v${version}/"
    
    log_success "Created release archive: dist/ims-v${version}.tar.gz"
    log_info "Release binaries built in: $build_dir"
}

# Build Docker image with version tag
build_docker_image() {
    local version=$1
    
    log_info "Building Docker image for version $version..."
    
    cd "$PROJECT_ROOT"
    docker build -t "ims:v$version" -t "ims:latest" .
    
    log_success "Built Docker image: ims:v$version"
}

# Create git tag
create_git_tag() {
    local version=$1
    local tag="v$version"
    
    log_info "Creating git tag $tag..."
    
    # Commit version changes
    git add VERSION go.mod CHANGELOG.md
    git commit -m "chore: release version $version"
    
    # Create annotated tag
    git tag -a "$tag" -m "Release version $version"
    
    log_success "Created git tag $tag"
    log_info "To push the tag, run: git push origin $tag"
}

# Show usage
show_usage() {
    echo "Usage: $0 [major|minor|patch|<version>]"
    echo ""
    echo "Examples:"
    echo "  $0 patch          # Increment patch version (1.0.0 -> 1.0.1)"
    echo "  $0 minor          # Increment minor version (1.0.1 -> 1.1.0)"
    echo "  $0 major          # Increment major version (1.1.0 -> 2.0.0)"
    echo "  $0 1.2.3          # Set specific version"
    echo ""
    echo "Options:"
    echo "  --docker-only     # Only build Docker image, skip binaries"
    echo "  --no-docker       # Skip Docker image build"
    echo "  --no-tag          # Skip git tag creation"
    echo "  --help            # Show this help"
}

# Main function
main() {
    local release_type="$1"
    local docker_only=false
    local no_docker=false
    local no_tag=false
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --docker-only)
                docker_only=true
                shift
                ;;
            --no-docker)
                no_docker=true
                shift
                ;;
            --no-tag)
                no_tag=true
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                if [ -z "$release_type" ]; then
                    release_type="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [ -z "$release_type" ]; then
        log_error "Please specify release type or version"
        show_usage
        exit 1
    fi
    
    # Checks
    check_git_repo
    check_working_directory
    
    # Get current and new version
    current_version=$(get_current_version)
    log_info "Current version: $current_version"
    
    if [[ "$release_type" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        new_version="$release_type"
        validate_version "$new_version"
    elif [[ "$release_type" =~ ^(major|minor|patch)$ ]]; then
        new_version=$(increment_version "$current_version" "$release_type")
    else
        log_error "Invalid release type: $release_type"
        show_usage
        exit 1
    fi
    
    log_info "New version: $new_version"
    
    # Confirm release
    echo -n "Proceed with release $new_version? [y/N] "
    read -r confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        log_info "Release cancelled"
        exit 0
    fi
    
    # Update version files
    update_version_file "$new_version"
    update_go_mod "$new_version"
    update_changelog "$new_version"
    
    # Build binaries and Docker image
    if [ "$docker_only" = true ]; then
        build_docker_image "$new_version"
    else
        if [ "$no_docker" = false ]; then
            build_docker_image "$new_version"
        fi
        build_release_binaries "$new_version"
    fi
    
    # Create git tag
    if [ "$no_tag" = false ]; then
        create_git_tag "$new_version"
    fi
    
    log_success "Release $new_version completed successfully!"
    
    echo ""
    echo "📦 Release Summary:"
    echo "   Version: $new_version"
    echo "   Binaries: dist/v$new_version/"
    echo "   Docker: ims:v$new_version"
    if [ "$no_tag" = false ]; then
        echo "   Git tag: v$new_version"
        echo ""
        echo "🚀 Next steps:"
        echo "   git push origin main"
        echo "   git push origin v$new_version"
        echo "   docker push your-registry/ims:v$new_version"
    fi
}

# Run main function with all arguments
main "$@" 
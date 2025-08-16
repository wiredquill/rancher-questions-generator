#!/bin/bash
# Optimized Build Script using Remote BuildKit Server
# Based on Docker Build Agent specifications

set -e

# Configuration
export BUILDKIT_HOST=tcp://10.0.10.120:1234
REGISTRY="ghcr.io/wiredquill"
PROJECT="rancher-questions-generator"
VERSION="${1:-1.4.0}"
TIMESTAMP=$(date +%Y%m%d-%H%M)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✅${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠️${NC} $1"
}

error() {
    echo -e "${RED}❌${NC} $1"
    exit 1
}

# Verify BuildKit connection
verify_buildkit() {
    log "Verifying BuildKit server connection..."
    if ! buildctl debug workers > /dev/null 2>&1; then
        warning "BuildKit server not available, falling back to local Docker"
        unset BUILDKIT_HOST
        return 1
    fi
    success "BuildKit server connected at $BUILDKIT_HOST"
    return 0
}

# Build backend with optimization
build_backend() {
    log "Building backend container..."
    
    local tags=(
        "${REGISTRY}/${PROJECT}-backend:${VERSION}"
        "${REGISTRY}/${PROJECT}-backend:latest"
        "${REGISTRY}/${PROJECT}-backend:claude-session-${TIMESTAMP}"
    )
    
    local output_args=""
    for tag in "${tags[@]}"; do
        output_args+="--output type=image,name=${tag},push=true "
    done
    
    if [[ -n "$BUILDKIT_HOST" ]]; then
        buildctl build \
            --frontend dockerfile.v0 \
            --local context=./backend \
            --local dockerfile=./backend \
            --opt filename=Dockerfile.optimized \
            ${output_args} \
            --export-cache type=registry,ref=${REGISTRY}/${PROJECT}-backend:buildcache,mode=max \
            --import-cache type=registry,ref=${REGISTRY}/${PROJECT}-backend:buildcache \
            --platform linux/amd64,linux/arm64
    else
        docker buildx build \
            --file ./backend/Dockerfile.optimized \
            --platform linux/amd64,linux/arm64 \
            --push \
            --cache-from type=registry,ref=${REGISTRY}/${PROJECT}-backend:buildcache \
            --cache-to type=registry,ref=${REGISTRY}/${PROJECT}-backend:buildcache,mode=max \
            $(printf -- "--tag %s " "${tags[@]}") \
            ./backend
    fi
    
    success "Backend built and pushed: ${tags[*]}"
}

# Build frontend with optimization
build_frontend() {
    log "Building frontend container..."
    
    local tags=(
        "${REGISTRY}/${PROJECT}-frontend:${VERSION}"
        "${REGISTRY}/${PROJECT}-frontend:latest"
        "${REGISTRY}/${PROJECT}-frontend:claude-session-${TIMESTAMP}"
    )
    
    local output_args=""
    for tag in "${tags[@]}"; do
        output_args+="--output type=image,name=${tag},push=true "
    done
    
    if [[ -n "$BUILDKIT_HOST" ]]; then
        buildctl build \
            --frontend dockerfile.v0 \
            --local context=./frontend-simple \
            --local dockerfile=./frontend-simple \
            --opt filename=Dockerfile.optimized \
            ${output_args} \
            --export-cache type=registry,ref=${REGISTRY}/${PROJECT}-frontend:buildcache,mode=max \
            --import-cache type=registry,ref=${REGISTRY}/${PROJECT}-frontend:buildcache \
            --platform linux/amd64,linux/arm64
    else
        docker buildx build \
            --file ./frontend-simple/Dockerfile.optimized \
            --platform linux/amd64,linux/arm64 \
            --push \
            --cache-from type=registry,ref=${REGISTRY}/${PROJECT}-frontend:buildcache \
            --cache-to type=registry,ref=${REGISTRY}/${PROJECT}-frontend:buildcache,mode=max \
            $(printf -- "--tag %s " "${tags[@]}") \
            ./frontend-simple
    fi
    
    success "Frontend built and pushed: ${tags[*]}"
}

# Validate images
validate_images() {
    log "Validating built images..."
    
    # Test backend
    log "Testing backend image..."
    if docker run --rm ${REGISTRY}/${PROJECT}-backend:${VERSION} --help > /dev/null 2>&1; then
        success "Backend image validation passed"
    else
        error "Backend image validation failed"
    fi
    
    # Test frontend
    log "Testing frontend image..."
    if docker run --rm -d --name test-frontend -p 18080:8080 ${REGISTRY}/${PROJECT}-frontend:${VERSION}; then
        sleep 5
        if curl -f http://localhost:18080/health > /dev/null 2>&1; then
            success "Frontend image validation passed"
        else
            error "Frontend image validation failed"
        fi
        docker stop test-frontend > /dev/null 2>&1
    else
        error "Frontend image failed to start"
    fi
}

# Image size analysis
analyze_images() {
    log "Analyzing image sizes..."
    
    echo "Backend Images:"
    docker images ${REGISTRY}/${PROJECT}-backend --format "table {{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
    
    echo ""
    echo "Frontend Images:"
    docker images ${REGISTRY}/${PROJECT}-frontend --format "table {{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
}

# Main execution
main() {
    log "Starting optimized build for ${PROJECT} v${VERSION}"
    
    # Check prerequisites
    if ! command -v buildctl &> /dev/null && ! command -v docker &> /dev/null; then
        error "Neither buildctl nor docker is available"
    fi
    
    # Verify BuildKit connection
    verify_buildkit || true
    
    # Build components
    build_backend
    build_frontend
    
    # Validate builds
    validate_images
    
    # Analyze results
    analyze_images
    
    success "Build completed successfully!"
    log "Images available:"
    log "  Backend: ${REGISTRY}/${PROJECT}-backend:${VERSION}"
    log "  Frontend: ${REGISTRY}/${PROJECT}-frontend:${VERSION}"
    log "  Session: claude-session-${TIMESTAMP}"
}

# Execute main function
main "$@"
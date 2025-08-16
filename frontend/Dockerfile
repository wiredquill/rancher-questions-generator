# syntax=docker/dockerfile:1.7
# Multi-stage build for optimized React frontend with security hardening
ARG NODE_VERSION=18
ARG ALPINE_VERSION=3.19
ARG NGINX_VERSION=1.25-alpine

# Builder stage with Node.js dependencies and cache optimization
FROM docker.io/node:${NODE_VERSION}-alpine${ALPINE_VERSION} AS builder

# Install security updates and build dependencies
RUN apk add --no-cache \
    dumb-init \
    && apk upgrade

# Create non-root user
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nextjs -u 1001

WORKDIR /app

# Copy package files and install dependencies (separate layer for caching)
COPY package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --only=production --ignore-scripts && \
    npm cache clean --force

# Copy source code and build
COPY . .
RUN npm run build

# Production stage with hardened nginx
FROM docker.io/nginx:${NGINX_VERSION} AS production

# Install security updates
RUN apk add --no-cache \
    dumb-init \
    && apk upgrade \
    && rm -rf /var/cache/apk/*

# Create nginx user and directories
RUN addgroup -g 101 -S nginx || true && \
    adduser -S -D -H -u 101 -h /var/cache/nginx -s /sbin/nologin -G nginx -g nginx nginx || true && \
    mkdir -p /var/cache/nginx /var/log/nginx /var/run/nginx && \
    chown -R nginx:nginx /var/cache/nginx /var/log/nginx /var/run/nginx /usr/share/nginx/html

# Copy built application
COPY --from=builder --chown=nginx:nginx /app/build /usr/share/nginx/html

# Copy optimized nginx configuration
COPY --chown=nginx:nginx nginx.conf /etc/nginx/conf.d/default.conf

# Create optimized nginx.conf with security headers
RUN cat > /etc/nginx/nginx.conf << 'EOF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx/nginx.pid;

events {
    worker_connections 1024;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    # Security headers
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data: https:; connect-src 'self';" always;
    
    # Performance optimizations
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    server_tokens off;
    
    # Compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
    
    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    
    include /etc/nginx/conf.d/*.conf;
}
EOF

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:80/ || exit 1

# Use non-root user
USER nginx

EXPOSE 80

STOPSIGNAL SIGQUIT

CMD ["nginx", "-g", "daemon off;"]

# Development stage for hot reload
FROM builder AS development

# Install development dependencies
RUN npm install

# Expose development server port
EXPOSE 3000

# Start development server
CMD ["npm", "start"]
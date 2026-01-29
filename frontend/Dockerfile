# Build stage
FROM node:20-alpine AS build

WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build for production
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy built assets from build stage
COPY --from=build /app/dist /usr/share/nginx/html

# Copy nginx template (will be processed at runtime)
COPY nginx.conf.template /etc/nginx/templates/default.conf.template

# Default backend URL for local docker-compose
ENV BACKEND_URL=http://backend:8080

EXPOSE 80

# nginx:alpine processes templates in /etc/nginx/templates/ automatically on startup

# syntax=docker/dockerfile:1

# ---- frontend build ----
FROM node:22-alpine AS frontend
WORKDIR /build
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- backend build ----
FROM golang:1.26-alpine AS backend
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /chatemp ./cmd/server

# ---- runtime ----
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 1000 chatemp \
    && mkdir -p /data /app \
    && chown chatemp:chatemp /data
WORKDIR /app
COPY --from=backend /chatemp /app/chatemp
COPY --from=frontend /build/dist /app/static
USER chatemp
EXPOSE 8080
ENTRYPOINT ["/app/chatemp"]

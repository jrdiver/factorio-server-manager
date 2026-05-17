# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /build
COPY package.json webpack.config.js tailwind.config.js ./
COPY ui/ ./ui/
COPY app/index.html ./app/index.html
RUN npm install && npm run build

# Stage 2: Build Go binary
FROM golang:1.21-alpine AS backend-builder
WORKDIR /build
COPY src/ ./src/
RUN cd src && CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o /build/factorio-server-manager .

# Stage 3: Runtime — glibc is required for Factorio Server binaries to run
FROM ubuntu:24.04

ENV FACTORIO_VERSION=stable \
    RCON_PASS="" \
    RCON_PORT=27015

VOLUME /opt/fsm-data /opt/factorio/saves /opt/factorio/mods /opt/factorio/config

EXPOSE 80/tcp 34197/udp 27015/tcp

RUN apt-get update && apt-get install -y curl tar xz-utils jq && rm -rf /var/lib/apt/lists/*

WORKDIR /opt
RUN mkdir -p /opt/fsm/app

# Copy built binary and frontend assets
COPY --from=backend-builder /build/factorio-server-manager /opt/fsm/factorio-server-manager
COPY --from=frontend-builder /build/app/ /opt/fsm/app/
COPY conf.json.example /opt/fsm/conf.json

COPY docker/entrypoint.sh /opt/entrypoint.sh
RUN sed -i 's/\r$//' /opt/entrypoint.sh && chmod +x /opt/entrypoint.sh

ENTRYPOINT ["/opt/entrypoint.sh"]

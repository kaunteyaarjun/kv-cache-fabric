# syntax=docker/dockerfile:1

# =============================================================================
# Stage 1: Build the Rust Tiered Memory Daemon
# =============================================================================
FROM rust:1.80-slim-bullseye AS rust-builder
WORKDIR /app

# Install protobuf compiler required by tonic-build
RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler \
    pkg-config \
    libssl-dev \
    && rm -rf /var/lib/apt/lists/*

COPY Cargo.toml build.rs main.rs ./
COPY proto/ proto/

RUN cargo build --release --bin memory-daemon

# =============================================================================
# Stage 2: Build the Go Disaggregated Conductor Proxy
# =============================================================================
FROM golang:1.22-bullseye AS go-builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o conductor .

# =============================================================================
# Target: memory-daemon (Standalone Rust gRPC Service)
# =============================================================================
FROM debian:bullseye-slim AS memory-daemon
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=rust-builder /app/target/release/memory-daemon /usr/local/bin/memory-daemon

EXPOSE 50051
ENV GRPC_BIND_ADDR=0.0.0.0:50051
CMD ["memory-daemon"]

# =============================================================================
# Target: conductor (Go Prefix-Caching & OpenAI-Compatible Reverse Proxy)
# =============================================================================
FROM debian:bullseye-slim AS conductor
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*

COPY --from=go-builder /app/conductor /usr/local/bin/conductor

EXPOSE 8080
ENV PORT=8080
ENV DAEMON_ADDR=memory-daemon:50051
CMD ["conductor"]

# =============================================================================
# Default Target: standalone (All-In-One Container for Quick Evaluation)
# =============================================================================
FROM debian:bullseye-slim AS standalone
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*

COPY --from=rust-builder /app/target/release/memory-daemon /usr/local/bin/memory-daemon
COPY --from=go-builder /app/conductor /usr/local/bin/conductor

EXPOSE 8080 50051

# Launch the Rust gRPC daemon in background, then start Go Conductor in foreground
RUN echo '#!/bin/sh\n\
memory-daemon &\n\
sleep 1\n\
export DAEMON_ADDR="127.0.0.1:50051"\n\
exec conductor\n' > /app/entrypoint.sh && chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]

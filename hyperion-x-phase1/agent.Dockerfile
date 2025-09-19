# --- Build stage ---
FROM rust:1.82 AS build
WORKDIR /app

# Copy manifest & pre-build dummy for caching
# The build context is './agent', so the path is just 'Cargo.toml'
COPY Cargo.toml .
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release || true

# Copy real sources and build
# The build context is './agent', so the path is just './src'
COPY src ./src
RUN cargo build --release

# --- Runtime stage ---
FROM debian:bookworm-slim AS runtime
WORKDIR /app

# Install minimal runtime deps
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libssl3 \
    libstdc++6 \
 && rm -rf /var/lib/apt/lists/*

# Copy binary
COPY --from=build /app/target/release/hyperion-agent /app/hyperion-agent

# Expose port
EXPOSE 9090

# Run agent
ENTRYPOINT ["/app/hyperion-agent"]
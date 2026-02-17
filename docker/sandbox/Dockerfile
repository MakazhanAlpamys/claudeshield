# ClaudeShield Sandbox Base Image
# Minimal, secure environment for running Claude Code agents

FROM ubuntu:24.04

LABEL maintainer="ClaudeShield <team@claudeshield.dev>"
LABEL org.opencontainers.image.source="https://github.com/claudeshield/claudeshield"

# Avoid interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Install essential dev tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    curl \
    wget \
    ca-certificates \
    build-essential \
    python3 \
    python3-pip \
    python3-venv \
    nodejs \
    npm \
    jq \
    ripgrep \
    fd-find \
    tree \
    && rm -rf /var/lib/apt/lists/*

# Install Go
RUN curl -fsSL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV PATH="/usr/local/go/bin:${PATH}"

# Create non-root user for sandbox
RUN groupadd -r sandbox && useradd -r -g sandbox -m -s /bin/bash sandbox

# Install ClaudeShield policy shell wrapper
COPY claudeshield-shell.sh /usr/local/bin/claudeshield-shell
RUN chmod +x /usr/local/bin/claudeshield-shell

# Create policy directory
RUN mkdir -p /etc/claudeshield && chown sandbox:sandbox /etc/claudeshield

# Set up workspace
RUN mkdir -p /workspace && chown sandbox:sandbox /workspace
WORKDIR /workspace

# Drop to non-root user
USER sandbox

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
    CMD test -d /workspace || exit 1

# Keep container running
CMD ["sleep", "infinity"]

FROM debian:bookworm-slim AS build
WORKDIR /src
RUN apt-get update && apt-get install -y curl imagemagick && rm -rf /var/lib/apt/lists/*
RUN curl https://mise.run | sh
ENV PATH="/root/.local/bin:${PATH}"
ENV MISE_TRUSTED_CONFIG_PATHS="/src"
ENV MISE_AUTO_INSTALL=false
COPY mise.toml ./
RUN mise install go
COPY ssh/go.mod ssh/go.sum ./ssh/
RUN cd ssh && mise exec -- go mod download
COPY ssh/ ./ssh/
COPY content/ ./content/
COPY config.yaml ./
RUN for f in $(find content/images -type f -size +5242880c 2>/dev/null); do \
      case "$f" in \
        *.jpg|*.jpeg) mogrify -resize '4096x4096>' -define jpeg:extent=4800KB "$f" ;; \
        *.png) mogrify -resize '4096x4096>' -quality 85 "$f" ;; \
        *.gif) mogrify -resize '4096x4096>' "$f" ;; \
      esac; \
    done
RUN cd ssh && CGO_ENABLED=0 mise exec -- go build -o /ssh.paullj.com ./cmd/ssh.paullj.com

FROM alpine:3.21
RUN apk add --no-cache chafa
WORKDIR /app
COPY --from=build /ssh.paullj.com /app/ssh.paullj.com
COPY config.yaml ./
COPY --from=build /src/content/ content/
ENV PAULLJ_SSH_HOST_KEY_PATH=/data/keys/id_ed25519
ENTRYPOINT ["/app/ssh.paullj.com"]

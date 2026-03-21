FROM debian:bookworm-slim AS build
WORKDIR /src
RUN apt-get update && apt-get install -y curl imagemagick chafa && rm -rf /var/lib/apt/lists/*
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
RUN cd ssh && CGO_ENABLED=0 mise exec -- go build -o /prerender-images ./cmd/prerender-images
RUN /prerender-images --out /app/image-cache --config config.yaml

FROM alpine:3.21
RUN apk add --no-cache chafa
WORKDIR /app
RUN adduser -D -h /app appuser
COPY --from=build /ssh.paullj.com /app/ssh.paullj.com
COPY --from=build /app/image-cache /app/image-cache
COPY config.yaml ./
COPY --from=build /src/content/ content/
RUN apk add --no-cache su-exec && \
    mkdir -p /data/keys && chown -R appuser:appuser /app /data/keys
COPY entrypoint.sh /app/entrypoint.sh
ENV PAULLJ_SSH_HOST_KEY_PATH=/data/keys/id_ed25519
ENV PAULLJ_SSH_IMAGES_CACHE_DIR=/app/image-cache
ENTRYPOINT ["/app/entrypoint.sh"]

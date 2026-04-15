# syntax=docker/dockerfile:1.7

FROM node:20-bookworm-slim AS frontend

WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY index.html postcss.config.js tailwind.config.js tsconfig.json webpack.config.js ./
COPY src ./src
RUN npm run typecheck && npm run build

FROM golang:1.24-bookworm AS backend

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend ./

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -trimpath -ldflags="-s -w" -o /out/modelrun ./cmd/modelrun

FROM python:3.11-slim-bookworm AS runtime

ARG MODELSCOPE_VERSION=1.35.4

ENV MODELRUN_ADDR=:8080 \
  MODELRUN_DATA=/var/lib/modelrun/modelrun.db \
  MODELRUN_STATIC_DIR=/app/dist \
  MODELSCOPE_CACHE=/var/lib/modelrun/modelscope \
  PYTHONUNBUFFERED=1 \
  PIP_NO_CACHE_DIR=1

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl git git-lfs \
  && git lfs install --system \
  && python -m pip install --upgrade pip \
  && python -m pip install "modelscope==${MODELSCOPE_VERSION}" \
  && modelscope --help >/dev/null \
  && rm -rf /var/lib/apt/lists/*

RUN useradd --create-home --shell /usr/sbin/nologin modelrun \
  && mkdir -p /app /var/lib/modelrun \
  && chown -R modelrun:modelrun /app /var/lib/modelrun

WORKDIR /app

COPY --from=backend /out/modelrun /app/modelrun
COPY --from=frontend /src/dist /app/dist
COPY docker/entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/modelrun /app/entrypoint.sh \
  && chown -R modelrun:modelrun /app

USER modelrun

EXPOSE 8080
VOLUME ["/var/lib/modelrun"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD curl -fsS http://127.0.0.1:8080/api/health >/dev/null || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]

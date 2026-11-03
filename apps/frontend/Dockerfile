# syntax=docker/dockerfile:1

# El contexto de build es la raíz del repo: el frontend vive en un workspace
# pnpm, así que necesita el lockfile y los scripts raíz.

# --- Base -----------------------------------------------------------------
FROM node:22-alpine AS base

WORKDIR /app

ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"

RUN corepack enable

# --- Dependencias ---------------------------------------------------------
FROM base AS deps

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/frontend/package.json apps/frontend/package.json

RUN pnpm install --frozen-lockfile --filter frontend...

# Contrato EF-12 y casos compartidos con Go, también usados por TypeScript.
COPY apps/backend/internal/submission/definition.json apps/backend/internal/submission/definition.json
COPY apps/backend/internal/submission/testdata/ apps/backend/internal/submission/testdata/

# --- Desarrollo ------------------------------------------------------------
FROM deps AS dev

COPY apps/frontend/ apps/frontend/

EXPOSE 5173

CMD ["pnpm", "--filter", "frontend", "dev", "--host", "0.0.0.0"]

# --- Compilación -----------------------------------------------------------
FROM deps AS build

COPY apps/frontend/ apps/frontend/

RUN pnpm --filter frontend build

# --- Producción ------------------------------------------------------------
FROM nginx:alpine AS prod

COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/apps/frontend/dist /usr/share/nginx/html

EXPOSE 80

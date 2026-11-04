# syntax=docker/dockerfile:1

# --- Etapa de compilación -------------------------------------------------
FROM golang:1.27-alpine AS build

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate
RUN mkdir -p /out/data && cp -r data/* /out/data/ 2>/dev/null || true

# --- Etapa del worker -----------------------------------------------------
# El worker cumple las colas en segundo plano sin servir tráfico HTTP. Puede
# escalar por separado de la API.
FROM gcr.io/distroless/static-debian12:nonroot AS worker

WORKDIR /app

COPY --from=build /out/worker /app/worker
COPY --from=build --chown=65532:65532 /out/data /app/data

USER nonroot:nonroot

ENTRYPOINT ["/app/worker"]

# --- Etapa de la API ------------------------------------------------------
# Última etapa: es el objetivo por defecto del build.
FROM gcr.io/distroless/static-debian12:nonroot AS api

WORKDIR /app

COPY --from=build /out/api /app/api
COPY --from=build /out/migrate /app/migrate
COPY --from=build --chown=65532:65532 /out/data /app/data

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/api"]

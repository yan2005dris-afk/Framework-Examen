# syntax=docker/dockerfile:1

# --- Etapa de compilación -------------------------------------------------
FROM golang:1.27-alpine AS build

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate
RUN mkdir -p /out/data

# --- Etapa de ejecución ---------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/api /app/api
COPY --from=build /out/migrate /app/migrate
COPY --from=build --chown=65532:65532 /out/data /app/data

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/api"]

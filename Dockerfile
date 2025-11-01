# syntax=docker/dockerfile:1
FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build API
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/api ./cmd/api

# Build Worker  
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/worker ./cmd/worker

# Build VAPID key generator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/vapidgen ./cmd/vapidgen

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /bin/api /bin/api
COPY --from=build /bin/worker /bin/worker
COPY --from=build /bin/vapidgen /bin/vapidgen
COPY db/migrations /app/migrations

EXPOSE 8080
USER nonroot
ENTRYPOINT ["/bin/api"]

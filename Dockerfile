# syntax=docker/dockerfile:1
FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod .
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api || true
RUN CGO_ENABLED=0 go build -o /bin/worker ./cmd/worker || true

FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=build /bin/api /bin/api
EXPOSE 8080
ENTRYPOINT ["/bin/api"]

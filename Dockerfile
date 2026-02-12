FROM golang:1.25 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o eco-api .

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/eco-api /eco-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/eco-api"]

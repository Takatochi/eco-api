FROM golang:1.25 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN C_GO_ENABLE=0 GOOS=linux go build -o eco-api ./...

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/eco-api /eco-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/eco-api"]
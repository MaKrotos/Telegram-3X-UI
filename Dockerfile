FROM golang:1.24.4-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o telegramxui ./cmd/telegramxui
RUN go install github.com/air-verse/air@latest
RUN go install github.com/pressly/goose/v3/cmd/goose@latest
COPY .air.toml .

FROM golang:1.24.4-alpine
WORKDIR /app
COPY --from=builder /app/telegramxui ./telegramxui
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /go/bin/air /usr/local/bin/air
CMD ["air"] 
FROM golang:1.25-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/marketnews ./cmd/marketnews

FROM alpine:3.20

RUN adduser -D app
USER app
WORKDIR /app

COPY --from=build /bin/marketnews /usr/local/bin/marketnews

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/marketnews"]

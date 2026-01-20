FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY . .
RUN if [ ! -f go.mod ]; then go mod init sono-version-service; fi && \
    go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/main .
RUN mkdir -p /app/data/apks
EXPOSE 80
CMD ["./main"]

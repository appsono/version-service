FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY . .
RUN rm -f go.mod go.sum
RUN go mod init sono-version-service
RUN go get github.com/aws/aws-sdk-go-v2@v1.24.0 && \
    go get github.com/aws/aws-sdk-go-v2/config@v1.26.1 && \
    go get github.com/aws/aws-sdk-go-v2/credentials@v1.16.12 && \
    go get github.com/aws/aws-sdk-go-v2/service/s3@v1.47.5 && \
    go get github.com/go-chi/chi/v5@v5.0.11 && \
    go get github.com/joho/godotenv@v1.5.1 && \
    go get github.com/lib/pq@v1.10.9
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/main .
RUN mkdir -p /app/data/apks
EXPOSE 80
CMD ["./main"]

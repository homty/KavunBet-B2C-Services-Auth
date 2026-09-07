FROM golang:1.26.5 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/wallet-service ./cmd/api



FROM scratch
WORKDIR /app
COPY --from=builder /out/wallet-service /app/wallet-service

#Prevents the app from running as root. Numeric IDs work even though scratch has no /etc/passwd.
USER 65532:65532
#Documents the HTTP/GRPC port; it does not publish it by itself.
EXPOSE 8083
EXPOSE 50052

ENTRYPOINT ["/app/wallet-service"]
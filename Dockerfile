FROM golang:1.15 as builder

# Set Environment Variables
ENV HOME /app
ENV CGO_ENABLED 0
ENV GOOS linux

WORKDIR /app
# COPY go.mod go.sum ./
# RUN go mod download
COPY . .

# Build the Go app
RUN go build -v -a -installsuffix cgo -o xscar cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/xscar .

CMD ["./xscar"]

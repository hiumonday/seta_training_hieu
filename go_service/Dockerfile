FROM golang:1.23-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
# conisder separating the build stage and the runtime stage for smaller image size
RUN go build -o server ./cmd/server

EXPOSE 8080

CMD ["./server"]
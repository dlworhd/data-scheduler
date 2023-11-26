FROM golang:latest

WORKDIR /app

COPY . .

RUN go build -o main.go .

CMD ["./main.go"]
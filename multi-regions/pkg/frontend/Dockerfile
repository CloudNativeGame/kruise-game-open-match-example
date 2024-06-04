FROM golang:alpine as go
WORKDIR /app
ENV GO111MODULE=on

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o frontend .

CMD ["/app/frontend"]

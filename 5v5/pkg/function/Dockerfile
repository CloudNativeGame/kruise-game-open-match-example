FROM golang:alpine as go
WORKDIR /app
ENV GO111MODULE=on

COPY . .
#RUN go mod edit -replace open-match.dev/open-match@v0.0.0-dev=open-match.dev/open-match@v1.7.0
#RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o matchfunction .

CMD ["/app/matchfunction"]
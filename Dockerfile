FROM golang:1.16-alpine

WORKDIR /go/src/app

COPY . .

EXPOSE 80

RUN apk add build-base

RUN go get -d -v ./...
RUN go build -o main -v

CMD ["./main"]
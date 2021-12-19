FROM golang:1.17.3-alpine3.14 as build

ENV GOPATH /go

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ARG VERSION
RUN mkdir /app
WORKDIR /app

COPY . .

RUN go build -ldflags="-X 'main.version=$VERSION'" main.go

FROM alpine:3.12

COPY --from=build app/main app/main

EXPOSE 8080

ENTRYPOINT /app/main
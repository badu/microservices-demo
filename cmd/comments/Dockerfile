FROM golang:1.20-alpine as builder

MAINTAINER Badu <badu@badu.ro>

RUN apk update \
    && apk add --no-cache ca-certificates git make

WORKDIR /comments_workdir

COPY . .

RUN CGO_ENABLED=0 go build -ldflags "-s -w" -v -o ./build/service github.com/badu/microservices-demo/cmd/comments

FROM alpine:3.17

RUN apk update \
    && apk add --no-cache ca-certificates tzdata && \
    addgroup -S badu && adduser -S badu -G badu

WORKDIR /app

COPY --from=builder /comments_workdir/build/service /app

USER badu

CMD /app/service

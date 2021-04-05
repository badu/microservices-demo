FROM golang:1.16

#ENV TZ=Europe/Bucharest
#RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENV config=docker

WORKDIR /app

COPY ../../api_gateway /app

RUN go mod download

RUN go get github.com/githubnemo/CompileDaemon

EXPOSE 5000

ENTRYPOINT CompileDaemon --build="go build cmd/gateway/main.go" --command=./main

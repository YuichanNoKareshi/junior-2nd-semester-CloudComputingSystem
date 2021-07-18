FROM golang:latest

WORKDIR $GOPATH/src/websocketserver
COPY . .
ENV GOPROXY https://goproxy.io,direct
RUN go mod tidy
RUN go build .

EXPOSE 8080 
ENTRYPOINT ["./websocketserver"]

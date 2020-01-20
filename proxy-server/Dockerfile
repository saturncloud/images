FROM golang:1.8-alpine

RUN apk --no-cache add git

WORKDIR /go/src/proxy-main
ADD ./go/proxy-main.go ./

RUN go get -v
RUN go install -v .

ENV PROXY_LISTEN_PORT=8080
ENV PROXY_FALLBACK_URL=http://localhost:$PROXY_LISTEN_PORT/fallback
ENV PROXY_RESOURCE_URL=http://localhost:80
ENV JWT_MINUTES_EXPIRE=60

EXPOSE 8080

CMD ["/go/bin/proxy-main"]


FROM golang:1.8-alpine

WORKDIR /go/src/proxy-main
COPY ./proxy-main/*.go .

RUN go get -d -v ./proxy-main/...
RUN go install -v ./proxy-main/...

ENV PROXY_LISTEN_PORT=8080
ENV PROXY_FALLBACK_URL=http://localhost:$PROXY_LISTEN_PORT/fallback
ENV PROXY_RESOURCE_URL=http://localhost:80
ENV JWT_MINUTES_EXPIRE=60

EXPOSE 8080


CMD ["proxy-main"]


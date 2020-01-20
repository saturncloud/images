FROM golang:1.13-alpine as builder

WORKDIR /app
ADD . .

RUN go mod download
RUN go build .

FROM alpine:latest

COPY --from=0 /app/proxy-server /app/proxy-server

ENV PROXY_LISTEN_PORT=8080
ENV PROXY_FALLBACK_URL=http://localhost:$PROXY_LISTEN_PORT/fallback
ENV PROXY_RESOURCE_URL=http://localhost:80
ENV JWT_MINUTES_EXPIRE=60

EXPOSE 8080

CMD ["/app/proxy-server"]

FROM golang:1.14.12 as builder

WORKDIR /go/src/github.com/Octops/gameserver-ingress-controller/examples/gameserver-http

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

# final image
FROM alpine:3.12

RUN adduser -D -u 1000 server
COPY --from=builder /go/src/github.com/Octops/gameserver-ingress-controller/examples/gameserver-http/server /home/server/server
RUN chown -R server /home/server && \
    chmod o+x /home/server/server

USER 1000
ENTRYPOINT ["/home/server/server"]
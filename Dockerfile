FROM golang:1.17 AS builder

WORKDIR /go/src/github.com/Octops/gameserver-ingress-controller

COPY . .

RUN make build && chmod +x /go/src/github.com/Octops/gameserver-ingress-controller/bin/octops-controller

FROM alpine:3.12

RUN addgroup -S octops \
    && adduser -S -g octops octops \
    && apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /go/src/github.com/Octops/gameserver-ingress-controller/bin/octops-controller /app/

RUN chown -R octops:octops ./

USER octops

ENTRYPOINT ["./octops-controller"]
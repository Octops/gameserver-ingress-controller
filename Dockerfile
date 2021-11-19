FROM golang:1.17 AS builder

WORKDIR /go/src/github.com/Octops/gameserver-ingress-controller

COPY . .

RUN make build && chmod +x /go/src/github.com/Octops/gameserver-ingress-controller/bin/octops-controller

FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /go/src/github.com/Octops/gameserver-ingress-controller/bin/octops-controller /app/

ENTRYPOINT ["./octops-controller"]
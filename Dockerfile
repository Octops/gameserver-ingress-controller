FROM --platform=$BUILDPLATFORM golang:1.19 AS builder

WORKDIR /go/src/github.com/Octops/gameserver-ingress-controller

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN make build && chmod +x /go/src/github.com/Octops/gameserver-ingress-controller/bin/octops-controller

FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /go/src/github.com/Octops/gameserver-ingress-controller/bin/octops-controller /app/

ENTRYPOINT ["./octops-controller"]
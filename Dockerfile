FROM golang:1.9-alpine as builder

RUN apk add -U git \
    && go get github.com/golang/dep/...

WORKDIR /go/src/github.com/evildecay/etcdkeeper

ADD src ./
ADD Gopkg.* ./

RUN dep ensure -update \
    && go build -o etcdkeeper.bin etcdkeeper/main.go

FROM alpine:3.7

RUN apk add --no-cache ca-certificates
RUN apk add --no-cache ca-certificates

WORKDIR /etcdkeeper
COPY --from=builder /go/src/github.com/evildecay/etcdkeeper/etcdkeeper.bin .
ADD assets assets

EXPOSE 8000

ENTRYPOINT ["./etcdkeeper.bin"]

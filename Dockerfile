FROM golang:1.12.10-alpine3.9 as builder

RUN apk add -U git \
    && go get github.com/golang/dep/...

WORKDIR /go/src/github.com/lixiangyun/etcdkeeper

ADD src ./
ADD Gopkg.* ./

RUN dep ensure -update \
    && go build -o etcdkeeper.bin etcdkeeper/main.go

FROM alpine:3.7

RUN apk add --no-cache ca-certificates
RUN apk add --no-cache ca-certificates

WORKDIR /etcdkeeper
COPY --from=builder /go/src/github.com/lixiangyun/etcdkeeper/etcdkeeper.bin .
ADD assets assets

EXPOSE 8000

ENTRYPOINT ["./etcdkeeper.bin"]

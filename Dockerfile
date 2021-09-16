FROM golang:1.19-alpine

WORKDIR /workdir
COPY *.go *.html *.js go.mod go.sum ./
RUN go mod tidy
RUN GCO_ENABLED=0 go build .

FROM alpine:latest
WORKDIR /root/
COPY --from=0 /workdir/shareonce /workdir/*.html /workdir/*.js ./

ENTRYPOINT ["./shareonce"]
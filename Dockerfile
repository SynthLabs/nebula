# STEP 1 build executable binary
FROM golang:alpine as builder
# Install SSL ca certificates
RUN apk update && apk add git && apk add ca-certificates

COPY . $GOPATH/src/github.com/synthlabs/nebula
WORKDIR $GOPATH/src/github.com/synthlabs/nebula
#get dependancies
RUN go get -u github.com/golang/dep/cmd/dep \
    && dep ensure
#build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/nebula ./cmd/nebula
# STEP 2 build a small image
# start from scratch
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy our static executable
COPY --from=builder /go/bin/nebula /go/bin/nebula

EXPOSE 7946

ENTRYPOINT ["/go/bin/nebula"]
# Build Go Server Binary
FROM golang:1.14.4

ARG VERSION

WORKDIR /project

# Only copy go.mod and go.sum, and download go mods separately to support layer caching
COPY ./go.* ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go install -v \
            -ldflags="-w -s -X main.version=${VERSION}" \
            .

# Build Docker with Only Server Binary
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=0 /go/bin/spinnaker-github-proxy /bin/server

RUN addgroup -g 1001 KeisukeYamashita && adduser -D -G KeisukeYamashita -u 1001 KeisueYamashita

USER 1001

CMD ["/bin/server"]

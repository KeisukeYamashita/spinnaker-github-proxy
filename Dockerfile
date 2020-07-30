# Build Go Server Binary
FROM golang:1.14.4

ARG GITHUB_TOKEN
ARG SERVICE_NAME
ARG VERSION

# GITHUB_TOKEN is used to fetch codes from private repository
RUN echo "machine github.com login $GITHUB_TOKEN" > ~/.netrc

WORKDIR /project

# Only copy go.mod and go.sum, and download go mods separately to support layer caching
COPY ./go.* ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go install -v \
            -ldflags="-w -s -X main.version=${VERSION} -X main.serviceName=${SERVICE_NAME}" \
            .

# Build Docker with Only Server Binary
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=0 /go/bin/gh-proxy-authn /bin/server

RUN addgroup -g 1001 keke && adduser -D -G keke -u 1001 keke

USER 1001

CMD ["/bin/server"]

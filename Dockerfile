FROM golang:1.15.2-alpine as builder

LABEL Maintainer="Evorts Technology"

RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

RUN adduser -D -g '' appuser

WORKDIR /apps/

COPY go.* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /go/bin/app .

FROM alpine:latest

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/bin/app /go/bin/app
COPY --from=builder /apps/config.docker.yml /go/bin/config.yml

ENV TZ=Asia/Jakarta

WORKDIR /go/bin/

USER appuser

CMD ["./app"]
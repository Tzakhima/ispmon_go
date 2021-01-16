FROM golang:1.15.6-alpine3.12 AS builder
WORKDIR /go/src/github.com/tzakhima/ispmon_go/
COPY ./*.go ./
COPY ./*.mod ./
RUN CGO_ENABLED=0 GOOS=linux  go build -o ispmon_go
CMD ["./ispmon_go", "-verbose", "true"]

FROM alpine:latest
WORKDIR /app
COPY --from=builder /go/src/github.com/tzakhima/ispmon_go/ispmon_go .
CMD  ["./ispmon_go"]

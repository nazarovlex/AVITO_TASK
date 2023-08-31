FROM golang:1.17-alpine as builder

WORKDIR /app
COPY . ./
RUN go build \
  -o bin/ \
  ./cmd

FROM alpine:latest

RUN addgroup -g 1001 -S app && mkdir -p /app && adduser -u 1001 -S -G app -h /app app
USER app
WORKDIR /app
COPY --from=builder --chown=app /app/bin/* /app/
COPY --from=builder --chown=app /app/reports /app/reports

CMD ["/app/cmd"]
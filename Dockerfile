FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

ARG CACHEBUST=1
ARG BUILD_TAGS=
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -tags "${BUILD_TAGS}" -ldflags="-s -w" -o gooodclass_server .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=builder /app/gooodclass_server /usr/local/bin/gooodclass_server

EXPOSE 8000

ENTRYPOINT ["gooodclass_server"]

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE
RUN case "$SERVICE" in tracker|ticket|wiki|buddy) ;; *) exit 1 ;; esac && \
    CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/$SERVICE && \
    CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -trimpath -o /out/seed ./cmd/seed && \
    CGO_ENABLED=0 go build -trimpath -o /out/wiki-index ./cmd/wiki-index

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S -G app app
COPY --from=build /out/ /app/
USER app
EXPOSE 8080
CMD ["/app/server"]

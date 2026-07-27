# builder
FROM --platform=$BUILDPLATFORM golang:1.24 AS builder
WORKDIR /app
ARG TARGETOS TARGETARCH

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /osrs-leaderboard

# runtime
FROM gcr.io/distroless/static-debian12
COPY --from=builder /osrs-leaderboard /osrs-leaderboard
USER nonroot:nonroot
ENTRYPOINT ["/osrs-leaderboard"]

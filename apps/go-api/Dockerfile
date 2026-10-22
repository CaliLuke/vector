FROM golang:1.27.3-bookworm AS build

WORKDIR /src

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -o /out/go-plane ./cmd/system && \
    CGO_ENABLED=0 go build -trimpath -o /out/go-bootstrap ./cmd/bootstrap && \
    CGO_ENABLED=0 go build -trimpath -o /out/go-background-worker ./cmd/background-worker && \
    CGO_ENABLED=0 go build -trimpath -o /out/go-support-migrate ./cmd/support-migrate && \
    CGO_ENABLED=0 go build -trimpath -o /out/river github.com/riverqueue/river/cmd/river

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/ /
COPY migrations/support/ /migrations/support/

EXPOSE 8080

ENTRYPOINT ["/go-plane"]

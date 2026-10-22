FROM golang:1.27.0-bookworm AS build
WORKDIR /src
ADD https://github.com/CaliLuke/autok-logal/releases/download/v0.5.0/autok-logal-0.5.0.tar.gz /tmp/logal.tar.gz
RUN echo '81aa7fb8527cac7ae037d9712b1b7e204300f7f525e51e59b19690840decc122  /tmp/logal.tar.gz' | sha256sum -c - \
    && tar -xzf /tmp/logal.tar.gz --strip-components=1 -C /src \
    && go build -trimpath -ldflags='-X main.version=0.5.0' -o /out/logal ./cmd/logal

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl lsof \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/logal /usr/local/bin/logal
ENTRYPOINT ["logal"]

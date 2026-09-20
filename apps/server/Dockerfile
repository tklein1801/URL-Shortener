FROM golang:1.27.1-alpine AS build
WORKDIR /src
ENV GOWORK=off
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server .
RUN mkdir -p /out/data && chmod 0700 /out/data

FROM scratch
WORKDIR /app
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build --chown=10001:10001 /out/data /data
COPY --from=build /out/server /server
USER 10001:10001
ENV MASTER_TOKEN_FILE=/data/master-token
ENV SQLITE_PATH=/data/links.db
VOLUME ["/data"]
EXPOSE 3000
ENTRYPOINT ["/server"]

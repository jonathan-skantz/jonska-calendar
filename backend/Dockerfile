FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd
RUN CGO_ENABLED=0 GOBIN=/out go install github.com/pressly/goose/v3/cmd/goose@v3.28.0

# One-shot migration runner. Goose reads its settings from the environment;
# set GOOSE_DBSTRING when running it.
FROM gcr.io/distroless/static-debian13:nonroot AS migrate
COPY --from=build /out/goose /usr/local/bin/goose
COPY db/migrations /migrations
ENV GOOSE_DRIVER=postgres GOOSE_MIGRATION_DIR=/migrations
ENTRYPOINT ["goose"]
CMD ["up"]

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]

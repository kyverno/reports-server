ARG ARCH
FROM golang:1.24 as build

WORKDIR /
COPY . ./

RUN GOOS=linux CGO_ENABLED=0 go build -ldflags="-w -s" -o reports-server ./main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build reports-server reports-server
COPY --from=build pkg/storage/db/migrations migrations
USER 65534
ENTRYPOINT ["/reports-server"]

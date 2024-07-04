ARG ARCH
FROM golang:1.22.2 as build

WORKDIR /
COPY . ./

RUN GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -o reports-server ./main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build reports-server reports-server
USER 65534
ENTRYPOINT ["/reports-server"]

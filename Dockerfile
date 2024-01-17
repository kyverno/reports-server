# Update the base image in Makefile when updating golang version. This has to
# be pre-pulled in order to work on GCB.
ARG ARCH
FROM golang:1.21.5 as build

WORKDIR /
COPY . ./
# RUN go mod download

# COPY pkg pkg
# COPY cmd cmd
# COPY Makefile Makefile

# ARG ARCH
# ARG GIT_COMMIT
# ARG GIT_TAG
RUN GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-w -s" -o policy-reports ./cmd/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build policy-reports policy-reports
USER 65534
ENTRYPOINT ["/policy-reports"]

FROM registry.hub.docker.com/library/golang:1.23.4-bookworm as build

WORKDIR /build

COPY go.mod .
COPY go.sum .

RUN #go list -e $(go list -f '{{.Path}}' -m all); exit 0

COPY main.go .
COPY pkg pkg


RUN go vet -v
# run only unit tests
RUN go test -v -tags "all,!integration"

# distroless images require coompilation with CGO disabled
RUN CGO_ENABLED=0 go install

FROM gcr.io/distroless/static-debian12

COPY --from=build /go/bin/release-analyzer /release-analyzer

CMD ["/release-analyzer"]

FROM golang:1.22-bookworm AS builder

RUN export GOBIN=$HOME/work/bin
WORKDIR /go/src/app
ADD . .
RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o main .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /go/src/app/main /app/
WORKDIR /app
USER 65532:65532
CMD ["./main"]
FROM node:20-alpine AS frontend-builder
COPY frontend-node/ /app/
WORKDIR /app
RUN npm install
RUN npm run build

FROM golang:1.22-bookworm AS node-builder
RUN export GOBIN=$HOME/work/bin
WORKDIR /go/src/app
COPY goshared/ /go/src/goshared
COPY node/ .
RUN go get -d -v ./...
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o main .

FROM gcr.io/distroless/static-debian12
COPY --from=node-builder /go/src/app/main /app/
COPY --from=frontend-builder /app/out/ /app/static/
WORKDIR /app
EXPOSE 8080
USER 65532:65532
CMD ["./main"]
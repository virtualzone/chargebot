FROM node:20-alpine AS frontend-builder
COPY frontend/ /app/
WORKDIR /app
RUN npm install
RUN npm run build

FROM golang:1.22-bookworm AS server-builder

# build libsodium (dep of libzmq)
WORKDIR /build
RUN wget https://github.com/jedisct1/libsodium/releases/download/1.0.19-RELEASE/libsodium-1.0.19.tar.gz
RUN tar -xzvf libsodium-1.0.19.tar.gz
WORKDIR /build/libsodium-stable
RUN ./configure --disable-shared --enable-static
RUN make -j`nproc`
RUN make install

# build libzmq (dep of zmq datastore)
WORKDIR /build
RUN wget https://github.com/zeromq/libzmq/releases/download/v4.3.5/zeromq-4.3.5.tar.gz
RUN tar -xvf zeromq-4.3.5.tar.gz
WORKDIR /build/zeromq-4.3.5
RUN ./configure --enable-static --disable-shared --disable-Werror
RUN make -j`nproc`
RUN make install

RUN export GOBIN=$HOME/work/bin
WORKDIR /go/src/app
ADD . .
RUN go get -d -v ./...
RUN CGO_ENABLED=1 CGO_LDFLAGS="-lstdc++" go build -ldflags="-w -s -extldflags=-static" -o main .

FROM gcr.io/distroless/base-debian12
COPY --from=server-builder /go/src/app/main /app/
COPY --from=frontend-builder /app/out/ /app/static/
WORKDIR /app
EXPOSE 8080
USER 65532:65532
CMD ["./main"]
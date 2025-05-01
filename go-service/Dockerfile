FROM golang:1.24

RUN apt-get update && \
    apt-get install -y gdal-bin libgdal-dev && \
    rm -rf /var/lib/apt/lists/*

ENV CGO_ENABLED=1 \
    PKG_CONFIG_PATH=/usr/lib/pkgconfig

WORKDIR /app

COPY . .

RUN go mod download
RUN go build -o main .

EXPOSE 50051

CMD ["./main"]
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o /out/protocollens-server ./cmd/protocollens-server

FROM debian:bookworm-slim
WORKDIR /app
ENV PROTOCOLLENS_ADDR=:8080
COPY --from=build /out/protocollens-server /usr/local/bin/protocollens-server
EXPOSE 8080
CMD ["protocollens-server"]

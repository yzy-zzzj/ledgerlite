FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/ledgerlite ./cmd/server

FROM gcr.io/distroless/base-debian12
COPY --from=build /bin/ledgerlite /ledgerlite
EXPOSE 8080
ENTRYPOINT ["/ledgerlite"]

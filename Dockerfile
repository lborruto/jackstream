FROM golang:1.24-alpine AS build
ENV GOTOOLCHAIN=local
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath -ldflags "-s -w" \
    -o /jackstream ./cmd/jackstream

FROM scratch
COPY --from=build /jackstream /jackstream
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 7000 7001
ENTRYPOINT ["/jackstream"]

FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /jackstream ./cmd/jackstream

FROM alpine:3
COPY --from=build /jackstream /jackstream
EXPOSE 7000 7001
ENTRYPOINT ["/jackstream"]

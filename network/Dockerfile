# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS test
WORKDIR /src
COPY go.mod ./
COPY internal ./internal
COPY cmd ./cmd
RUN go test ./...

FROM test AS build
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/cholonet ./cmd/network

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /out/cholonet /cholonet
EXPOSE 8080
ENTRYPOINT ["/cholonet"]

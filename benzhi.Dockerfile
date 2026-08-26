FROM golang:1.23.12 AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web
RUN CGO_ENABLED=0 go build -mod=vendor -o /out/waterplant ./cmd/server

FROM golang:1.23.12
ENV GOPROXY=off GOSUMDB=off
WORKDIR /app
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web
COPY --from=build /out/waterplant ./waterplant
EXPOSE 18080
CMD ["/app/waterplant"]

FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/fake-gk ./cmd/fake-gk

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

ENV GIN_MODE=release
WORKDIR /app

COPY --from=build /out/fake-gk ./fake-gk
COPY src/public ./src/public
COPY web/templates ./web/templates

USER app
EXPOSE 3000

CMD ["./fake-gk"]

FROM golang:1.27.1-alpine AS builder

ARG BIN=server
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/${BIN} ./cmd/${BIN}

FROM gcr.io/distroless/static-debian12

ARG BIN=server
COPY --from=builder /out/${BIN} /scheduler
ENTRYPOINT ["/scheduler"]
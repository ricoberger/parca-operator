FROM golang:1.22.3 as builder
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
RUN CGO_ENABLED=0 go build -a -o manager main.go

FROM alpine:3.20.0
RUN apk update && apk add --no-cache ca-certificates
WORKDIR /
COPY --from=builder /workspace/manager .
USER nobody
ENTRYPOINT ["/manager"]

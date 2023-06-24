# syntax=docker/dockerfile:1

FROM golang:1.20-alpine as build-stage

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY *.go ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/proxy

FROM alpine:3.18 AS build-release-stage

WORKDIR /app

COPY --from=build-stage /app/proxy /app/proxy

EXPOSE 3000

ENTRYPOINT ["/app/proxy"]
FROM golang:1.26-alpine AS build
ENV GOWORK=off
WORKDIR /src/kubilitics-agent
# Build the agent in standalone module mode (GOWORK=off) so we don't need
# to drag the whole workspace (kubilitics-backend, kubilitics-cli) into the
# image just to satisfy go.work.
COPY kubilitics-agent/go.mod kubilitics-agent/go.sum ./
RUN go mod download
COPY kubilitics-agent/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/agent ./cmd/agent

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/agent /agent
USER nonroot:nonroot
ENTRYPOINT ["/agent"]

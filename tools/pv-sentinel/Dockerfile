# Build stage for the sentinel binary.
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sentinel ./cmd/sentinel

# Final stage.
FROM alpine:3.20

# open-iscsi is the package that provides iscsiadm, the binary the resize
# remediation execs (ISCSIADM_PATH defaults to /usr/sbin/iscsiadm). It is
# architecture-gated because Alpine ships open-iscsi for x86_64 only while CI
# builds linux/amd64 and linux/arm64 — an unconditional add breaks the arm64
# build. The cluster is amd64, so every node that needs the fix gets it; on
# arm64 the binary is simply absent and the remediation reports ErrNoISCSIADM
# instead of pretending it ran. The base image itself is unchanged.
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "amd64" ]; then \
        apk --no-cache add ca-certificates open-iscsi; \
    else \
        apk --no-cache add ca-certificates; \
    fi && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /home/appuser/

COPY --from=builder /app/sentinel .

RUN chown -R appuser:appgroup /home/appuser/

EXPOSE 8080

# The image is non-root per house convention, but the DaemonSet overrides this
# with runAsUser: 0 — probing a root-owned CSI mount directory requires write
# permission the appuser would not have. The image default stays non-root so a
# plain `docker run` of this image is not accidentally privileged.
USER appuser

CMD ["./sentinel"]

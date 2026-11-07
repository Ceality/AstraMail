# syntax=docker/dockerfile:1

# AstraMail 运行镜像不编译源码，直接下载 GitHub Release 中的 Linux 静态二进制。
# 构建时可通过 --build-arg VERSION=0.1.0 指定发布版本。

ARG VERSION=0.1.0
ARG REPO=Ceality/AstraMail

# ---- 下载并校验 Release 产物 ----
FROM alpine:3.22 AS fetch

ARG VERSION
ARG REPO

RUN apk add --no-cache ca-certificates curl

WORKDIR /out

RUN set -eux; \
    asset="astramail-${VERSION}-linux-amd64"; \
    base="https://github.com/${REPO}/releases/download/v${VERSION}"; \
    curl -fsSL --retry 3 -o "${asset}" "${base}/${asset}"; \
    curl -fsSL --retry 3 -o checksums.txt "${base}/checksums.txt"; \
    expected="$(awk -v f="${asset}" '$2 == f || $2 == "*" f { print $1 }' checksums.txt)"; \
    test -n "${expected}"; \
    echo "${expected}  ${asset}" | sha256sum -c -; \
    mv "${asset}" /out/astramail; \
    chmod 0755 /out/astramail

# ---- 最小运行镜像：静态二进制 + CA 证书，无 shell、无包管理器 ----
FROM scratch

ARG VERSION

LABEL org.opencontainers.image.title="AstraMail" \
      org.opencontainers.image.description="自动化多邮箱管理与聚合系统" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.source="https://github.com/Ceality/AstraMail"

COPY --from=fetch /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=fetch /out/astramail /astramail

ENV SERVER_HOST=0.0.0.0 \
    SERVER_PORT=3001

EXPOSE 3001

# 非 root 运行；scratch 中没有 /etc/passwd，使用数字 UID/GID。
USER 65532:65532

ENTRYPOINT ["/astramail"]

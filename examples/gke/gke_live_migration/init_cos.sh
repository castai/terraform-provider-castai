#!/bin/bash
set -euo pipefail

# Detect system architecture
ARCH=$(uname -m)
case "$ARCH" in
x86_64) ARCH="amd64" ;;
aarch64) ARCH="arm64" ;;
arm64) ARCH="arm64" ;;
amd64) ARCH="amd64" ;;
*)
  echo "Warning: Unsupported architecture: $ARCH, defaulting to amd64" >&2
  ARCH="amd64"
  ;;
esac

CRI_URL=https://storage.googleapis.com/castai-node-components/castai-cri-proxy/releases/0.27.0

wget ${CRI_URL}/castai-cri-proxy-linux-${ARCH}.tar.gz -O /var/tmp/castai-cri-proxy-linux-${ARCH}.tar.gz
wget ${CRI_URL}/castai-cri-proxy_SHA256SUMS -O /var/tmp/proxy_SHA256SUMS
SHA256_AMD64_FROM_FILE=$(head -n 1 /var/tmp/proxy_SHA256SUMS | awk '{print $1}')
SHA256_ARM64_FROM_FILE=$(sed -n '2p' /var/tmp/proxy_SHA256SUMS | awk '{print $1}')
pushd /var/tmp
sha256sum --ignore-missing --check /var/tmp/proxy_SHA256SUMS
popd
tar -xvzf /var/tmp/castai-cri-proxy-linux-${ARCH}.tar.gz -C /home/kubernetes/bin/ cri-proxy
chmod +x /home/kubernetes/bin/cri-proxy

cat <<EOF >/var/tmp/pre-install.yaml
packages:
    cri-proxy:
        downloadURL: ${CRI_URL}
        unpackDir: /home/kubernetes/bin
        arch:
            amd64:
                fileName: castai-cri-proxy-linux-amd64.tar.gz
                sha256sum: ${SHA256_AMD64_FROM_FILE}
            arm64:
                fileName: castai-cri-proxy-linux-arm64.tar.gz
                sha256sum: ${SHA256_ARM64_FROM_FILE}
EOF

sudo /home/kubernetes/bin/cri-proxy install --base-config=gke-cos --config /var/tmp/pre-install.yaml --debug 2>&1 | sudo tee /var/tmp/LIVE_INSTALL_LOG >/dev/null


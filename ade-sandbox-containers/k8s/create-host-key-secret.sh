#!/usr/bin/env bash
# Create the `cmux-ssh-host-key` Secret holding cmux's sshd host keypair.
#
# The host key is generated ONCE and reused forever. If the Secret already
# exists the script leaves it untouched — rotating the host key on every apply
# is exactly what forced clients to `ssh-keygen -R '[<lb-ip>]:2222'` after
# every pod restart (the key lived in the ephemeral container layer, since only
# ~/workspace is on the PV). Storing it here makes the fingerprint stable
# across restarts and rollouts.
#
# The private key is secret material, so it lives only in the Secret — never in
# the repo. The value is never echoed; only the Secret name is printed.
#
#   ./k8s/create-host-key-secret.sh
#
# To rotate deliberately:
#   kubectl -n ade-sandbox delete secret cmux-ssh-host-key && ./k8s/create-host-key-secret.sh
set -euo pipefail

NS="${NS:-ade-sandbox}"
NAME="cmux-ssh-host-key"

# Generate-once: an existing Secret wins, so re-running never rotates the key.
if kubectl -n "$NS" get secret "$NAME" >/dev/null 2>&1; then
  echo "secret/$NAME already exists in $NS — leaving host key untouched."
  echo "(to rotate: kubectl -n $NS delete secret $NAME, then re-run)"
  exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
ssh-keygen -q -t ed25519 -N '' -f "$tmp/host_ed25519_key"

kubectl -n "$NS" create secret generic "$NAME" \
  --from-file=host_ed25519_key="$tmp/host_ed25519_key" \
  --from-file=host_ed25519_key.pub="$tmp/host_ed25519_key.pub" >/dev/null

echo "created secret/$NAME in $NS (ed25519 host keypair)"

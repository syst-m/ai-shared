#!/usr/bin/env bash
# Create/refresh the `ade-agent-keys` Secret used by both sandbox StatefulSets.
#
# Nothing key-shaped is committed to this repo, so the Secret is applied
# imperatively. Values are taken from the environment if set, otherwise parsed
# out of ~/.zshrc (the active, uncommented `export` line wins).
#
#   ./k8s/create-agent-secret.sh
#   ANTHROPIC_API_KEY=sk-... ./k8s/create-agent-secret.sh   # override one value
#
# Keys consumed by the StatefulSets: ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN,
# OPENAI_API_KEY. Re-running is idempotent (server-side apply).
set -euo pipefail

NS="${NS:-ade-sandbox}"
ZRC="${ZSHRC:-$HOME/.zshrc}"

# Last active `export NAME=` line, minus the assignment and any wrapping quotes.
from_zshrc() {
  [ -f "$ZRC" ] || return 0
  grep -E "^[[:space:]]*export $1=" "$ZRC" | tail -1 \
    | sed -E "s/^[[:space:]]*export $1=//; s/^\"//; s/\"$//; s/^'//; s/'$//"
}

pick() { local v="${!1:-}"; [ -n "$v" ] && printf '%s' "$v" || from_zshrc "$1"; }

ANTHROPIC_API_KEY="$(pick ANTHROPIC_API_KEY)"
ANTHROPIC_AUTH_TOKEN="$(pick ANTHROPIC_AUTH_TOKEN)"
OPENAI_API_KEY="$(pick OPENAI_API_KEY)"

missing=0
for k in ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN OPENAI_API_KEY; do
  if [ -z "${!k}" ]; then
    echo "WARN: $k is empty (not in env, not an active export in $ZRC)" >&2
    missing=1
  fi
done
if [ "$missing" = 1 ]; then
  echo "Refusing to create a Secret with empty credentials. Set the missing" >&2
  echo "values in the environment or in $ZRC and re-run." >&2
  exit 1
fi

# Values are passed via the environment, never on the command line, so they do
# not appear in the process list or in shell history.
export ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN OPENAI_API_KEY

kubectl -n "$NS" create secret generic ade-agent-keys \
  --from-literal=ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" \
  --from-literal=ANTHROPIC_AUTH_TOKEN="$ANTHROPIC_AUTH_TOKEN" \
  --from-literal=OPENAI_API_KEY="$OPENAI_API_KEY" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "applied Secret ade-agent-keys in $NS (keys: ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, OPENAI_API_KEY)"

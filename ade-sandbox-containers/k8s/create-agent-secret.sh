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
# OPENAI_API_KEY, GH_TOKEN. Re-running is idempotent (server-side apply).
#
# GH_TOKEN is sourced differently: on macOS the gh token lives in the Keychain
# and ~/.config/gh/hosts.yml carries no token key, so `gh auth token` is the
# only non-interactive source. It is soft-required — an empty value warns
# instead of failing, because both StatefulSets mark the key `optional: true`.
set -euo pipefail

NS="${NS:-ade-sandbox}"
ZRC="${ZSHRC:-$HOME/.zshrc}"

# Last active `export NAME=` line, minus the assignment and any wrapping quotes.
from_zshrc() {
  [ -f "$ZRC" ] || return 0
  grep -E "^[[:space:]]*export $1=" "$ZRC" | tail -1 \
    | sed -E "s/^[[:space:]]*export $1=//; s/^\"//; s/\"$//; s/^'//; s/'$//"
}

# A value in ~/.zshrc can itself be a shell reference — OPENAI_API_KEY is
# exported as "${ANTHROPIC_AUTH_TOKEN}". A plain grep captures that literally,
# so follow a lone ${NAME} to its value (env first, then ~/.zshrc) up to a few
# hops. A literal value falls straight through the while test untouched.
resolve() {
  local v="$1" name next hops=0
  while [[ "$v" == '${'* ]] && (( hops < 5 )); do
    [[ "$v" =~ ^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$ ]] || break
    name="${BASH_REMATCH[1]}"
    next="$(pick "$name")"
    [ -z "$next" ] && break
    v="$next"; hops=$((hops + 1))
  done
  printf '%s' "$v"
}

# Env value wins; otherwise read from ~/.zshrc; then expand any ${NAME} reference.
pick() {
  local v="${!1:-}"
  [ -n "$v" ] || v="$(from_zshrc "$1")"
  resolve "$v"
}

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

# gh's token is not in ~/.zshrc: on macOS it lives in the Keychain and
# ~/.config/gh/hosts.yml carries no token key, so `gh auth token` is the only
# non-interactive source. Soft-required — both StatefulSets mark the key
# `optional: true`, so an empty value warns instead of aborting.
GH_TOKEN="${GH_TOKEN:-${GITHUB_TOKEN:-}}"
if [ -z "$GH_TOKEN" ] && command -v gh >/dev/null 2>&1; then
  GH_TOKEN="$(gh auth token 2>/dev/null || true)"
fi
if [ -z "$GH_TOKEN" ]; then
  echo "WARN: GH_TOKEN is empty (not in env, \`gh auth token\` returned nothing)." >&2
  echo "      gh in the sandboxes will be unauthenticated; continuing." >&2
fi

# The values are quoted into argv here, so they are briefly visible in the
# process list for the duration of the call; keeping them out of shell history
# is the reason this script exists instead of typing the values inline.
args=(
  --from-literal=ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY"
  --from-literal=ANTHROPIC_AUTH_TOKEN="$ANTHROPIC_AUTH_TOKEN"
  --from-literal=OPENAI_API_KEY="$OPENAI_API_KEY"
)
if [ -n "$GH_TOKEN" ]; then
  args+=(--from-literal=GH_TOKEN="$GH_TOKEN")
fi

kubectl -n "$NS" create secret generic ade-agent-keys "${args[@]}" \
  --dry-run=client -o yaml | kubectl apply -f -

keys="ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, OPENAI_API_KEY"
if [ -n "$GH_TOKEN" ]; then
  keys="$keys, GH_TOKEN"
fi
echo "applied Secret ade-agent-keys in $NS (keys: $keys)"

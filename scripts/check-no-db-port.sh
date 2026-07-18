#!/bin/bash
# Fails if the resolved docker-compose config publishes a host port for the
# db service. Postgres runs with sslmode=disable because it's reachable only
# over the internal app_net network — no firewall backstop protects it if
# that ever changes (Docker's own iptables rules bypass ufw's default-deny,
# confirmed empirically 2026-07-18). See
# docs/security-review/RISK-MITIGATION-PLAN-2026-07-18.md, Risk 1.
set -euo pipefail

cd "$(dirname "$0")/.."

# Required secrets have no defaults in the compose files (${VAR:?...}), so
# `docker compose config` fails to resolve without them. Real values aren't
# needed just to inspect the resolved port mappings.
export DB_PASSWORD="${DB_PASSWORD:-dummy-for-config-check}"
export JWT_SECRET="${JWT_SECRET:-dummy-for-config-check}"
export SECURITY_ENCRYPTION_KEY="${SECURITY_ENCRYPTION_KEY:-dummy-for-config-check}"

resolved_json=$(docker compose -f docker-compose.yml -f docker-compose.prod.yml config --format json)

port_count=$(echo "$resolved_json" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(len(d["services"]["db"].get("ports") or []))')

if [ "$port_count" != "0" ]; then
    echo "ERROR: db service has a ports: mapping in the resolved docker-compose config." >&2
    echo "Postgres must never be published to the host — see docs/security-review/RISK-MITIGATION-PLAN-2026-07-18.md, Risk 1." >&2
    echo "$resolved_json" | python3 -c 'import json,sys; print(json.dumps(json.load(sys.stdin)["services"]["db"].get("ports"), indent=2))' >&2
    exit 1
fi

echo "OK: db service has no published ports."

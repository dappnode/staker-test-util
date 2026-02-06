#!/usr/bin/env bash
set -euo pipefail

if [ -z "${DISCORD_STAKERS_TESTS_WEBHOOK:-}" ]; then
  echo "DISCORD_STAKERS_TESTS_WEBHOOK not set — skipping"
  exit 0
fi

: "${RESULT:?}"
: "${TEST_TYPE:?}"
: "${REPO:?}"
: "${SERVER_URL:?}"
: "${RUN_ID:?}"

PR_NUMBER="${PR_NUMBER:-}"

RESULT_UPPER=$(echo "$RESULT" | tr '[:lower:]' '[:upper:]')
if [ "$RESULT" = "success" ]; then
  COLOR=3066993
  EMOJI="✅"
else
  COLOR=15158332
  EMOJI="❌"
fi

RUN_URL="${SERVER_URL}/${REPO}/actions/runs/${RUN_ID}"
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

DESC="${EMOJI} **${TEST_TYPE} test — ${RESULT_UPPER}**"
[ -n "$PR_NUMBER" ] && DESC+=$'\n'"**PR:** [#${PR_NUMBER}](${SERVER_URL}/${REPO}/pull/${PR_NUMBER})"

jq -n \
  --arg title "$REPO" \
  --arg desc "$DESC" \
  --arg url "$RUN_URL" \
  --argjson color "$COLOR" \
  --arg ts "$TIMESTAMP" \
  '{
    embeds: [{
      title: $title,
      description: $desc,
      url: $url,
      
      color: $color,
      timestamp: $ts,
      footer: { text: "Staker CI · hoodi" }
    }]
  }' | curl -sS -f -H "Content-Type: application/json" -d @- "$DISCORD_STAKERS_TESTS_WEBHOOK" \
    || echo "::warning::Failed to post to Discord"

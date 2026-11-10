#!/usr/bin/env bash
# Regenerates the recordings in docs/media from the tapes in docs/media/tapes.
#
# Every command in a recording runs against the fixture backend (testdata/fixture), never against the Mail app.
# The tapes run in /tmp/mailslot-demo so that no personal path, account or configuration can appear on screen.
#
# Usage: docs/media/record.sh [tape name...]    e.g. docs/media/record.sh triage
set -euo pipefail

root=$(cd "$(dirname "$0")/../.." && pwd)
demo=/tmp/mailslot-demo

for program in go vhs jq; do
  command -v "$program" >/dev/null || { echo "record.sh: $program is not installed" >&2; exit 1; }
done

rm -rf "$demo"
mkdir -p "$demo/bin" "$demo/home" "$demo/work"
(cd "$root" && go build -o "$demo/bin/mailslot" .)
ln -s "$root/testdata/fixture" "$demo/fixture"

cat > "$demo/env.sh" <<ENV
unset MAILSLOT_ACCOUNT MAILSLOT_READ_ONLY MAILSLOT_NO_SEND MAILSLOT_FIXTURE_FAULT
# A separate HOME and cache directory keep the recording off the real lock file and configuration.
export HOME="$demo/home"
export PATH="$demo/bin:\$PATH"
export MAILSLOT_FIXTURE="$demo/fixture"
export MAILSLOT_FIXTURE_STATE="$demo/state.json"
export XDG_CONFIG_HOME="$demo/config"
export XDG_CACHE_HOME="$demo/cache"
export TZ=UTC
cd "$demo/work"
ENV

# Refuse to record unless the fixture backend is the one answering.
if ! (source "$demo/env.sh" && mailslot doctor | grep -q "fixture backend at $demo/fixture"); then
  echo "record.sh: the fixture backend is not active; not recording" >&2
  exit 1
fi

if [ "$#" -gt 0 ]; then
  names=("$@")
else
  names=(read search triage safety)
fi

cd "$root"
for name in "${names[@]}"; do
  # Each tape starts from the untouched fixture. All but the first start with Work as the default account.
  rm -rf "$demo/state.json" "$demo/config"
  if [ "$name" != read ]; then
    mkdir -p "$demo/config/mailslot"
    echo "account = 'Work'" > "$demo/config/mailslot/config.toml"
  fi
  vhs "docs/media/tapes/$name.tape"
done

rm -rf "$demo"

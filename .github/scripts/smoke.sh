#!/usr/bin/env bash
# Drive a built pickle binary through a fresh in-tree install: install, hooks, a ticket
# created and moved, board audit, doctor, and the pre-commit guard refusing tickets/ on a
# feature branch. Used by ci.yml's windows-smoke job (T-143) — the one check that runs the
# real pickle.exe on Windows — and runnable as-is on macOS/Linux.
#
# Usage: smoke.sh <bin-dir> <work-dir>
#   <bin-dir>  — directory holding the pickle (or pickle.exe) binary; put first on PATH,
#                because the git hooks call `pickle` by name
#   <work-dir> — an empty or absent directory; the child is named after its basename
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <bin-dir> <work-dir>" >&2
  exit 2
fi
export PATH="$1:$PATH"
mkdir -p "$2"
cd "$2"
child=$(basename "$PWD")

git init -q -b main
git -c user.name=ci -c user.email=ci@example.com commit -q --allow-empty -m init
pickle install --in-tree --agent claude
pickle hooks install
pickle ticket new "smoke" --project "$child"
pickle ticket move T-001 dropped --reason smoke
pickle board audit
pickle doctor

git checkout -q -b feat/T-001-smoke
git add tickets
# A failed commit proves nothing on its own (a hook that crashes fails it too), so require
# the guard's own refusal text.
if out=$(git -c user.name=ci -c user.email=ci@example.com commit -q -m x 2>&1); then
  echo "::error::pre-commit hook did not refuse tickets/ on a feature branch" >&2
  exit 1
fi
if ! grep -q "refusing to commit ticket bookkeeping on a feature branch" <<<"$out"; then
  echo "::error::commit failed, but not by the pre-commit guard:" >&2
  echo "$out" >&2
  exit 1
fi
echo "smoke: ok"

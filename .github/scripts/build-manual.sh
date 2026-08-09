#!/usr/bin/env bash
# Build the PDF/EPUB user manual via the native asciidoctor toolchain (snowball).
# Shared by .github/workflows/release.yml and .github/workflows/manual-smoke.yml so this
# logic exists in exactly one place (T-087) — it had drifted across five separate edits to
# release.yml's inline step before being extracted here.
#
# Usage: build-manual.sh <src-dir> <out-dir>
#   <src-dir> — directory containing snowball.yaml (checked out docs tree)
#   <out-dir> — where snowball writes pickle-user-manual.{pdf,epub}
#
# Runnable on a developer's macOS box, not just the Linux CI runner (T-087 decision 7): it
# only loads brew's shellenv as a fallback when brew is not already on PATH, and never
# hardcodes a Linuxbrew path outside that fallback.
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <src-dir> <out-dir>" >&2
  exit 2
fi
src_dir=$1
out_dir=$2

# Quiet brew's progress/hint chatter (its terminal-width probing is what triggers the
# intermittent "Broken pipe" failures on CI) and bypass Homebrew's Tap-Trust gate for our own
# tap (codcod/tap) — without it, a CI box refuses to install from it ("the following taps are
# not trusted"), leaving snowball uninstalled. Defaults only: a caller's environment (e.g.
# release.yml's own env: block) wins.
: "${HOMEBREW_NO_REQUIRE_TAP_TRUST:=1}"
: "${HOMEBREW_NO_AUTO_UPDATE:=1}"
: "${HOMEBREW_NO_ENV_HINTS:=1}"
: "${HOMEBREW_NO_COLOR:=1}"
export HOMEBREW_NO_REQUIRE_TAP_TRUST HOMEBREW_NO_AUTO_UPDATE HOMEBREW_NO_ENV_HINTS HOMEBREW_NO_COLOR

# 1. Homebrew's shellenv (decision 7): only when brew isn't already on PATH. The CI runner
# ships Homebrew preinstalled but off PATH by default; a developer's shell has usually already
# loaded it, and hardcoding the Linuxbrew path there would be wrong on macOS.
if ! command -v brew >/dev/null 2>&1; then
  if [ -x /home/linuxbrew/.linuxbrew/bin/brew ]; then
    eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
  else
    echo "build-manual: brew not found on PATH and no Linuxbrew fallback at" \
      "/home/linuxbrew/.linuxbrew/bin/brew — install Homebrew first." >&2
    exit 1
  fi
fi

# 2. The runner image's preinstalled Homebrew is a point-in-time snapshot and can lag behind
# homebrew-core's bottle metadata. When it does, a bottle built with a newer install-step DSL
# (observed: ruby's post-install step "remove") makes that stale brew abort with "unknown
# install step" instead of installing it -- a Homebrew-version mismatch, not a transient
# network blip, so the retry loop below alone never recovers from it (T-086). `brew update`
# (unlike `install`/`upgrade`) always fetches regardless of HOMEBREW_NO_AUTO_UPDATE, which only
# suppresses the *implicit* update those two commands would otherwise trigger.
brew update --quiet

# 3. Fully qualify the formula: a bare `snowball` resolves to homebrew/core's stemmer of the
# same name, not our tap's tool. Retry the install: it occasionally dies with a transient
# broken pipe mid-download. Skip entirely when snowball is already on PATH (a developer's box).
if ! command -v snowball >/dev/null 2>&1; then
  for attempt in 1 2 3; do
    brew install codcod/tap/snowball && break
    echo "brew install failed (attempt $attempt); retrying..."
    sleep 5
  done
fi

# 4. Diagnostics — always printed, never fatal. No prior CI run recorded this ground truth,
# which is exactly why the original bundle-not-found failure (T-087) took four runs to
# root-cause instead of one.
echo "--- build-manual: diagnostics ---"
brew --prefix ruby 2>&1 || true
type -a ruby gem bundle 2>&1 || true
ls -l "$(brew --prefix)/bin/bundle" "$(brew --prefix)/bin/bundler" 2>&1 || true
snowball doctor 2>&1 || true
echo "--- end diagnostics ---"

# 5. TEMPORARY bundler shim (T-087 decision 1). Homebrew's `ruby` formula is not keg-only and
# its keg does ship bin/bundle — what ruby's post_install removes is bundler from the *linked*
# prefix (HOMEBREW_PREFIX/bin/bundle and the prefix gem dir). So on a fresh runner `bundle` is
# missing from PATH while $(brew --prefix ruby)/bin still has it. That is measured, not assumed:
# this script's own diagnostics in manual-smoke run 31318708209 show `type -a bundle` -> not
# found and HOMEBREW_PREFIX/bin/bundle absent, yet prepending the keg's bin dir alone resolved
# bundle 4.0.16 — the `gem install` fallback below never fired there, and is therefore untested
# in CI. The real fix belongs in snowball itself (SNOW-002, tracked on the `unity` workspace's
# board: snowball's own Setup() should preflight/bootstrap bundler instead of a bare PATH
# lookup). Remove this whole block once a snowball release's `setup` makes `bundle` resolvable
# on the *caller's* PATH, not merely inside its own subprocess env — SNOW-002's sketch may only
# do the latter. Both halves are skipped outright when `bundle` already resolves, so a
# developer's own ruby (mise, rbenv, …) is never silently shadowed by Homebrew's.
if ! command -v bundle >/dev/null 2>&1; then
  ruby_bin="$(brew --prefix ruby)/bin"
  if [ -d "$ruby_bin" ]; then
    export PATH="$ruby_bin:$PATH"
  fi
fi
if ! command -v bundle >/dev/null 2>&1; then
  echo "build-manual: bundle not on PATH after ruby's bin dir; bootstrapping via gem install" \
    "(temporary shim — see SNOW-002)"
  gem install --no-document bundler
  gem_bindir="$(gem environment gemdir)/bin"
  export PATH="$gem_bindir:$PATH"
fi

# 6. v0.2.2's manual build failed with npm's UNABLE_TO_GET_ISSUER_CERT_LOCALLY against
# Homebrew's node, which ships with no CA bundle configured — deterministic, not a flake, and
# OS-level (snowball's own toolchain boundary, D2, leaves CA trust to the environment).
if [ -f /etc/ssl/certs/ca-certificates.crt ]; then
  export NODE_EXTRA_CA_CERTS=/etc/ssl/certs/ca-certificates.crt
fi

# 7. Pre-setup assertions (decision 8) — unguarded: a failure here must stop the script with
# the missing precondition named, rather than being masked by `snowball setup`'s own error.
ruby --version
gem --version
bundle --version

# 8. Run snowball itself from the docs tree.
(
  cd "$src_dir"
  snowball setup
  snowball doctor
  snowball build -o "$out_dir"
)

#!/usr/bin/env python3
# Target metric (2026-09-29 retro, T-139 baseline): of the applicability-gate runs whose verdict had
# no blocking finding, how many stopped for the human before work started, and how many of those
# stops got only a bare "go". Claude Code transcripts only.
# usage: python3 2026-09-29-gate-stops.py SINCE UNTIL   (dates YYYY-MM-DD, inclusive)
# for the human before starting work, and was the human's answer a bare "go"?
import json, glob, os, re, sys
root = os.path.join(os.path.expanduser(os.environ.get("CLAUDE_CONFIG_DIR", "~/.claude")), "projects")
since, until = sys.argv[1], sys.argv[2]
BARE = re.compile(r"^\W*(go|go ahead|go on|approved?|approve|yes|y|ok|okay|proceed|continue|lgtm|1|do it|sounds good|agreed)\W*$", re.I)
NOISE = ("<task-notification>", "Another Claude session sent a message", "<command-name>", "<bash-input>", "<bash-stdout>", "<local-command")
WORK = re.compile(r"git (checkout -b|switch -c)|pickle ticket move \S+ (in-development|development)")
BLOCK = re.compile(r"(\d+|one|two|three)\s+blocking|blocking (finding|problem|issue)s?\s*[:(]|verdict[^\n]{0,40}\bblock", re.I)
NONBLOCK = re.compile(r"(no|0|zero|none)\s+blocking|blocking[^\n]{0,15}(none|0\b)", re.I)
def texts(c):
    return [c] if isinstance(c, str) else [y.get("text", "") for y in c or [] if isinstance(y, dict) and y.get("type") == "text"]
runs = []
for f in glob.glob(root + "/*/*.jsonl"):
    L = []
    for line in open(f, errors="ignore"):
        try: L.append(json.loads(line))
        except Exception: pass
    for i, d in enumerate(L):
        if d.get("type") != "assistant": continue
        for x in (d.get("message") or {}).get("content") or []:
            if not (isinstance(x, dict) and x.get("type") == "tool_use" and x.get("name") in ("Agent", "Task")
                    and re.search("pplicab", json.dumps(x.get("input", {})))): continue
            ts = d.get("timestamp", "")
            if not (since <= ts[:10] <= until): continue
            verdict = None; outcome = "session-ended"; reply = ""
            for e in L[i + 1:]:
                raw = json.dumps(e)
                if verdict is None:
                    # sync tool result, a background agent's hand-back (isMeta), or its completion notification
                    c = (e.get("message") or {}).get("content") if isinstance(e.get("message"), dict) else None
                    if (isinstance(c, list) and any(isinstance(y, dict) and y.get("tool_use_id") == x["id"] and "Async agent launched" not in json.dumps(y) for y in c)) \
                       or ("Subagent hand-back" in raw and e.get("type") == "user") or (x["id"] in raw and "<task-notification>" in raw and "completed" in raw):
                        verdict = raw
                    continue
                if e.get("type") == "user" and not e.get("isMeta"):
                    c = (e.get("message") or {}).get("content")
                    t = re.sub(r"<system-reminder>.*?</system-reminder>", "", " ".join(texts(c)), flags=re.S).strip()
                    if not t or t.startswith(NOISE): continue
                    outcome = "stopped"; reply = t; break
                if e.get("type") == "assistant" and WORK.search(json.dumps((e.get("message") or {}).get("content"))):
                    outcome = "proceeded"; break
            if verdict is None: outcome = "no-verdict"
            blocking = bool(verdict and BLOCK.search(verdict) and not NONBLOCK.search(verdict))
            bare = outcome == "stopped" and bool(BARE.match(reply))
            runs.append((ts[:10], os.path.basename(f)[:8], "BLOCKING" if blocking else "clean", outcome, "bare-go" if bare else reply[:50].replace("\n", " ")))
runs.sort()
for r in runs: print(*r, sep=" | ")
clean = [r for r in runs if r[2] == "clean" and r[3] in ("stopped", "proceeded")]
stopped = [r for r in clean if r[3] == "stopped"]
bare = [r for r in stopped if r[4] == "bare-go"]
print(f"\ngate runs {len(runs)} | blocking {sum(r[2]=='BLOCKING' for r in runs)} | clean & decided {len(clean)}")
print(f"clean runs that stopped for the human: {len(stopped)} ({len(stopped)/max(len(clean),1):.0%}); of those, answered with a bare go: {len(bare)}")

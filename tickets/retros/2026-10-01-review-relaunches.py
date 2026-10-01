#!/usr/bin/env python3
# Target metric (T-138 baseline): of the brine reviews that concluded (a `pickle ticket move <ID>
# done|rework` tool call) within 2 hours of that ticket's implement or rework session handing it to
# review (`pickle ticket move <ID> in-review`), how many were relaunched — their session opened with
# the human's validate/review trigger as its first non-noise message. A review chained into the
# implement/rework session is not relaunched. Also prints the guard: delegated-reviewer sub-agents
# (first prompt carries the spawn template's role clause) that ran a ticket move, commit or branch
# switch. Claude Code transcripts only.
# usage: python3 2026-10-01-review-relaunches.py SINCE UNTIL   (dates YYYY-MM-DD, inclusive)
#        python3 2026-10-01-review-relaunches.py SINCE UNTIL --show SID8   (print that session's cycle)
# Validate by hand on three cycles before quoting the number.
import json, glob, os, re, sys, datetime
root = os.path.join(os.path.expanduser(os.environ.get("CLAUDE_CONFIG_DIR", "~/.claude")), "projects")
since, until = sys.argv[1], sys.argv[2]
show = sys.argv[sys.argv.index("--show") + 1] if "--show" in sys.argv else None
MOVE = re.compile(r"\bpickle ticket move\s+([A-Z]+-\d+)\s+(\S+)")
TRIGGER = re.compile(r"\b(validate|review) ticket\s+([A-Z]+-\d+)", re.I)
NOISE = ("<task-notification>", "Another Claude session sent a message", "<command-name>", "<command-message>", "<bash-input>", "<bash-stdout>", "<local-command", "Caveat:")
ROLE = "You are a delegated reviewer (step 0 of this skill's"
FORBIDDEN = re.compile(r"\bpickle ticket move\b|\bgit (commit|checkout|switch)\b")
WINDOW = datetime.timedelta(hours=2)
def texts(c):
    return [c] if isinstance(c, str) else [y.get("text", "") for y in c or [] if isinstance(y, dict) and y.get("type") == "text"]
def human(d):
    if d.get("type") != "user" or d.get("isMeta") or d.get("isSidechain"): return None
    t = re.sub(r"<system-reminder>.*?</system-reminder>", "", " ".join(texts((d.get("message") or {}).get("content"))), flags=re.S).strip()
    return None if not t or t.startswith(NOISE) else t
def when(ts): return datetime.datetime.fromisoformat(ts.replace("Z", "+00:00"))
def kind(status):
    s = status.strip("'\"").lower()
    return "review" if "review" in s else "done" if "done" in s else "rework" if "rework" in s else None
def load(f):
    L = []
    for line in open(f, errors="ignore"):
        try: L.append(json.loads(line))
        except Exception: pass
    return L
handoffs, concluded, seen = {}, [], set()
for f in glob.glob(root + "/*/*.jsonl"):
    proj, sid = os.path.basename(os.path.dirname(f)), os.path.basename(f)[:8]
    L = load(f)
    first = next((t for t in map(human, L) if t is not None), "")
    for d in L:
        if d.get("type") != "assistant" or d.get("isSidechain"): continue
        ts = d.get("timestamp", "")
        for x in (d.get("message") or {}).get("content") or []:
            if not (isinstance(x, dict) and x.get("type") == "tool_use" and x.get("name") == "Bash"): continue
            for tid, status in MOVE.findall((x.get("input") or {}).get("command", "")):
                k = kind(status)
                if (proj, tid, ts[:16], k) in seen or k is None: continue  # resumed sessions copy history
                seen.add((proj, tid, ts[:16], k))
                if k == "review": handoffs.setdefault((proj, tid), []).append((ts, sid))
                elif since <= ts[:10] <= until: concluded.append((ts, proj, tid, k, sid, first))
rows, counted = [], set()
for ts, proj, tid, k, sid, first in sorted(concluded):
    prior = [(t, s) for t, s in handoffs.get((proj, tid), []) if t < ts and when(ts) - when(t) <= WINDOW]
    if not prior: continue
    hts, hsid = max(prior)
    if (proj, tid, hts) in counted: continue  # one review per hand-off; a retried move is not a second
    counted.add((proj, tid, hts))
    tm = TRIGGER.search(first)
    relaunched = hsid != sid and bool(tm) and tm.group(2).upper() == tid
    rows.append((ts[:16], proj.replace("-Users-nka-Projects-", "")[-24:], tid, k, f"{hsid}->{sid}", "RELAUNCHED" if relaunched else "chained" if hsid == sid else "other"))
    if show and sid.startswith(show):
        print(f"--- {tid}: handed to review {hts} in {hsid}; concluded {k} {ts} in {sid}\nfirst human message: {first[:200]!r}\n")
for r in rows: print(*r, sep=" | ")
rel = [r for r in rows if r[5] == "RELAUNCHED"]
print(f"\nconcluded reviews within 2h of a hand-off in {since}..{until}: {len(rows)}")
print(f"relaunched: {len(rel)} ({len(rel)/max(len(rows),1):.0%}); chained: {sum(r[5]=='chained' for r in rows)}; other: {sum(r[5]=='other' for r in rows)}")
guard = 0
for f in glob.glob(root + "/*/*/subagents/*.jsonl"):
    L = load(f)
    if not L or not any(ROLE in t for t in texts((L[0].get("message") or {}).get("content"))): continue
    ts = L[0].get("timestamp", "")
    if not (since <= ts[:10] <= until): continue
    for d in L:
        for x in ((d.get("message") or {}).get("content") or []) if d.get("type") == "assistant" else []:
            if isinstance(x, dict) and x.get("type") == "tool_use" and FORBIDDEN.search((x.get("input") or {}).get("command", "") or ""):
                guard += 1; print(f"GUARD: {f}: {x['input']['command'][:120]!r}")
print(f"guard: delegated-reviewer tool calls that moved, committed or switched branch: {guard}")

#!/usr/bin/env python3
# Target metric (T-137 baseline): of the out-of-band code-review runs typed after a brine review
# had reached a verdict, how many reported at least one correctness finding. Also prints the
# guard's baseline: median token use of brine validate/review sessions. Claude Code transcripts only.
# usage: python3 2026-10-01-code-review-after-validate.py SINCE UNTIL   (dates YYYY-MM-DD, inclusive)
# A run counts as "after validate" when the same project folder saw a validate/review trigger in
# the 48 hours before it. Validate by hand on three runs before quoting the number.
import json, glob, os, re, sys, statistics, datetime
root = os.path.join(os.path.expanduser(os.environ.get("CLAUDE_CONFIG_DIR", "~/.claude")), "projects")
since, until = sys.argv[1], sys.argv[2]
RUN = re.compile(r"^/code-review|<command-name>/code-review")  # same detector as 2026-09-29-weekly.py
TRIGGER = re.compile(r"\b(validate|review) ticket\s+([A-Z]+-\d+)", re.I)
NOISE = ("<task-notification>", "Another Claude session sent a message", "<bash-input>", "<bash-stdout>", "<local-command")
WINDOW = datetime.timedelta(hours=48)
def texts(c):
    return [c] if isinstance(c, str) else [y.get("text", "") for y in c or [] if isinstance(y, dict) and y.get("type") == "text"]
def human(d):
    if d.get("type") != "user" or d.get("isMeta"): return None
    t = re.sub(r"<system-reminder>.*?</system-reminder>", "", " ".join(texts((d.get("message") or {}).get("content"))), flags=re.S).strip()
    return None if not t or t.startswith(NOISE) else t
def when(ts): return datetime.datetime.fromisoformat(ts.replace("Z", "+00:00"))
triggers, runs, validate_tokens = {}, [], []
for f in glob.glob(root + "/*/*.jsonl"):
    proj = os.path.basename(os.path.dirname(f))
    L = []
    for line in open(f, errors="ignore"):
        try: L.append(json.loads(line))
        except Exception: pass
    first_human, seen, tokens = None, set(), 0
    for i, d in enumerate(L):
        ts = d.get("timestamp", "")
        m = d.get("message") if isinstance(d.get("message"), dict) else {}
        if d.get("type") == "assistant" and m.get("id") not in seen:
            seen.add(m.get("id"))
            u = m.get("usage") or {}
            tokens += sum(u.get(k, 0) or 0 for k in ("input_tokens", "output_tokens", "cache_creation_input_tokens", "cache_read_input_tokens"))
        t = human(d)
        if t is None: continue
        if first_human is None: first_human = t
        tm = TRIGGER.search(t)
        if tm and not RUN.search(t): triggers.setdefault(proj, []).append((ts, tm.group(2).upper()))
        if not (RUN.search(t) and since <= ts[:10] <= until): continue
        total = correct = 0; method = "none"; last_text = ""
        for e in L[i + 1:]:
            if human(e) is not None and not RUN.search(human(e)): break
            for x in ((e.get("message") or {}).get("content") or []) if isinstance(e.get("message"), dict) else []:
                if not isinstance(x, dict): continue
                if x.get("type") == "tool_use" and x.get("name") == "ReportFindings":
                    fs = (x.get("input") or {}).get("findings") or []
                    method = "tool"; total += len(fs); correct += sum(1 for y in fs if (y.get("category") or "").lower() == "correctness")
                if e.get("type") == "assistant" and x.get("type") == "text": last_text = x.get("text", "")
        if method == "none" and last_text:
            method = "text"; correct = len(re.findall(r"\bcorrectness\b", last_text, re.I))
        runs.append((ts, proj, os.path.basename(f)[:8], method, total, correct))
    if first_human and TRIGGER.search(first_human) and since <= (L[0].get("timestamp") or "")[:10] <= until:
        validate_tokens.append(tokens)
rows = []
for ts, proj, sid, method, total, correct in sorted(runs):
    prior = [(t, tid) for t, tid in triggers.get(proj, []) if t < ts and when(ts) - when(t) <= WINDOW]
    if not prior: continue
    rows.append((ts[:16], proj.replace("-Users-nka-Projects-", "")[-24:], sid, max(prior)[1], method, total, correct))
for r in rows: print(*r, sep=" | ")
hit = [r for r in rows if r[6] > 0]
print(f"\ncode-review runs after a validate in {since}..{until}: {len(rows)} (of {len(runs)} runs in window)")
print(f"runs with >= 1 correctness finding: {len(hit)} ({len(hit)/max(len(rows),1):.0%}); method tool={sum(r[4]=='tool' for r in rows)} text={sum(r[4]=='text' for r in rows)} none={sum(r[4]=='none' for r in rows)}")
if validate_tokens:
    print(f"guard: validate/review sessions {len(validate_tokens)}, median tokens {statistics.median(validate_tokens):,.0f}")

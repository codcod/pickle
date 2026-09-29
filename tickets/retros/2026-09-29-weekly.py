#!/usr/bin/env python3
# Per-ISO-week view of Claude Code sessions (2026-09-29 retro): brine use, dominant model, and the
# recurring things the human typed by hand. Pattern counts are regex matches on human messages —
# indicative, not exact; validate a sample before quoting one as a baseline.
# usage: python3 2026-09-29-weekly.py
import json, glob, os, re, datetime
from collections import Counter, defaultdict
root = os.path.join(os.path.expanduser(os.environ.get("CLAUDE_CONFIG_DIR", "~/.claude")), "projects")
wk = defaultdict(Counter); models = defaultdict(Counter); projs = defaultdict(set)
PAT = {
 "code-review": re.compile(r"^/code-review|<command-name>/code-review"),
 "pr-merged": re.compile(r"\b(pr|mr)\s*#?\d+\s+(is\s+)?merged|merged,? record", re.I),
 "trigger": re.compile(r"\b(implement|validate|review|rework|refine) ticket\b", re.I),
 "model-switch": re.compile(r"<command-name>/model"),
 "go-ahead": re.compile(r"^(go|go ahead|approved?|yes|ok|1|proceed)\.?$", re.I),
 "rules-complaint": re.compile(r"CLAUDE\.md|disobey|you keep", re.I),
}
for f in glob.glob(root + "/*/*.jsonl"):
    first = None; brine = False; c = Counter(); m = Counter()
    for line in open(f, errors="ignore"):
        try: d = json.loads(line)
        except Exception: continue
        ts = d.get("timestamp")
        if ts and not first: first = ts
        if "brine" in line: brine = True
        if d.get("type") == "assistant":
            mod = (d.get("message") or {}).get("model")
            if mod and not mod.startswith("<"): m[mod] += 1
        if d.get("type") == "user" and not d.get("isMeta"):
            cont = (d.get("message") or {}).get("content")
            texts = [cont] if isinstance(cont, str) else [x.get("text", "") for x in cont or [] if isinstance(x, dict) and x.get("type") == "text"]
            for t in texts:
                t = re.sub(r"<system-reminder>.*?</system-reminder>", "", t, flags=re.S).strip()
                for k, p in PAT.items():
                    if p.search(t): c[k] += 1
    if not first: continue
    y, w, _ = datetime.date.fromisoformat(first[:10]).isocalendar()
    key = f"{y}-W{w:02d}"
    wk[key]["sessions"] += 1; wk[key]["brine"] += brine; wk[key].update(c)
    if m: models[key][m.most_common(1)[0][0].replace("claude-", "")] += 1
    projs[key].add(os.path.basename(os.path.dirname(f)).replace("-Users-nka-Projects-", "").split("-")[-1])
cols = ["sessions", "brine", "trigger", "code-review", "pr-merged", "go-ahead", "model-switch", "rules-complaint"]
print("week     " + " ".join(f"{c[:8]:>8}" for c in cols) + "  models (sessions by dominant model)")
for k in sorted(wk):
    print(f"{k} " + " ".join(f"{wk[k][c]:>8}" for c in cols) + "  " + ", ".join(f"{n} {v}" for n, v in models[k].most_common(3)))

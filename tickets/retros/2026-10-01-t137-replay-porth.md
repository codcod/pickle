# T-137 replay — porth `8c214c8` (POR-002, PR 7 review tip)

Raw output of the new review-protocol step 3 angle hunt (no host tool), run 2026-10-01 by three
fresh angle sub-agents confined to a throwaway worktree, briefed only with step 3, the ticket as it
stood at review (porth-umbrella `b5ee3aa`), the diff `8c214c8~1..8c214c8`, the child's commands
and their angle group. Complexity `medium`: three sub-agents. Judged before any re-verification.

**Pass bar (T-137 Task 6): met, all three known bugs found.**

| POR-013 item | found as | evidence |
|---|---|---|
| 1 — early multi-part receipt dropped, message stranded `sent` | C1 | reproduced, 3 of 3 runs |
| 3 — `_drop()` discards acked, queued receipts | B1 (**blocking**), C4 | B1 reproduced (fails as is, passes with the fix); C4 `unreproduced:` with the traced race |
| 4 — `expand_dlr_url` mangles percent-encoding | A2, C3 | reproduced on the wire (A2) and in-process (C3) |

Also found: PR 7 finding 8 (negative `dlr-mask`, C2, reproduced). Every `correctness` row carries
a reproduction or an `unreproduced:` reason. A2 and C1 note the bug sits in a confirmed decision
or a gate-accepted risk, the same deference the smppai replay showed.

### Group A — angles 1 (caller contracts), 6 (breaking changes), 8 (confirmed decisions)

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| A1 | non-blocking | test-gap | — | Task 3 changed the contract between `kannel_send_sms` and `DLRHandler`: `protocol_data['dlr_mask']` went from a str with a `'1'` default to an int, and absent or non-integer masks now become 0. No test goes through the endpoint to check this. `test_kannel_api.py` never checks `dlr_mask`. `test_receipt_calls_kannel_dlr_url` and `test_dlr.py` build `protocol_data={'dlr_mask': 1}` by hand, so they skip the parser. If the old line comes back, every test still passes, and at runtime `'3' & 1` raises `TypeError` in `_callback` after the status has already changed. `_consume` swallows that, so the callback is silently lost. | Mutation: `sed` put back `protocol_data={'dlr_mask': params.get('dlr-mask', '1')}` in `src/porth/protocols/kannel/api.py`, then `uv run pytest -q` gave `78 passed`. File restored afterwards. | Add endpoint tests in `test_kannel_api.py` for `dlr-mask=3` giving int 3, mask absent giving 0, and `dlr-mask=abc` giving 0. Or make one integration test send its Kannel message through `/cgi-bin/sendsms`. |
| A2 | non-blocking | correctness | — | Decision 7's single-pass regex `%([dIFAtT])` also matches percent-escapes that are already in the `dlr-url`. Escapes such as `%A9`, `%F0`–`%FF` and lowercase `%d0`–`%df` occur in any UTF-8-encoded query value (`é` = `%C3%A9`). The code follows the decision exactly, so the defect is in the plan. Kannel's own escape expansion may behave the same way, but I could not check that offline. | See the A2 reproduction below. `?n=%C3%A9&id=%I` was sent as `/dlr?n=%C3id:s1%20stat:DELIVRD9&id=m-1`, and the server decoded `n` as `'�id:s1 stat:DELIVRD9'`. `?x=%d0` was sent as `?x=10`. | Note it under the Delivery reports section of `kannel-api.adoc`: a `%` followed by `d I F A t T` is always treated as an escape code, so encode such bytes some other way. If Kannel turns out not to do this, re-plan the substitution, for example by skipping `%` followed by two hex digits for `%A`/`%F`/`%d` (the ticket owner decides). |
| A3 | non-blocking | docs-gap | — | The ticket's Docs update requires the umbrella `design.md` §3/§4.2/§6 to change from "POST dlr-url" to "GET dlr-url", the version bump, and the `kannel-features.md` HTTPS row, all committed on the umbrella's `main`. The ticket's last History line says these edits were "left uncommitted". Until they are committed, the design document contradicts the code, which uses `self._session.get(url)` in `src/porth/core/dlr.py`, and the user manual, which now says GET. | From the ticket History (2026-09-26, IN DEVELOPMENT → IN REVIEW): "umbrella doc edits (design.md 1.8 GET, review-addendum v2.5, kannel-features HTTPS row) left uncommitted". The umbrella files were outside the review directory. | Commit the umbrella edits on its `main` before closing the ticket, keeping them separate from the other author's uncommitted 1.7 MO draft, or record a deferral. |

A2 reproduction (scratch script `porth/t137a_wire.py`, deleted afterwards; `uv run python t137a_wire.py`): expands each template with `expand_dlr_url` and sends a GET through `aiohttp.ClientSession` to a `TestServer` that records `raw_path` and `query`:
```
?id=%I&p=%p&a=%A -> ?id=m-1&p=%p&a=id%3As1%20stat%3ADELIVRD | wire: ('/dlr?id=m-1&p=%25p&a=id:s1%20stat:DELIVRD', {'id': 'm-1', 'p': '%p', 'a': 'id:s1 stat:DELIVRD'})
?n=%C3%A9&id=%I -> ?n=%C3id%3As1%20stat%3ADELIVRD9&id=m-1 | wire: ('/dlr?n=%C3id:s1%20stat:DELIVRD9&id=m-1', {'n': '�id:s1 stat:DELIVRD9', 'id': 'm-1'})
?x=%d0 -> ?x=10 | wire: ('/dlr?x=10', {'x': '10'})
```
Checked with no finding: callers and cross-module effects; confirmed decisions 1–8 and tasks 2/4 hold in the code. `make test` 82 passed, `make lint` clean.

### Group B — angles 2 (error, cancellation and shutdown), 5 (resource lifetimes), 7 (security)

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| B1 | blocking | correctness | — | Receipts that arrive while porth is closing a bind get acknowledged to the SMSC and then dropped. `SMPPClient._drop()` cancels the `_inbound` consumer first and only then awaits `_close(client)`, which sends `unbind` and waits for `unbind_resp`. A `deliver_sm` crossing the unbind is acked by smppai's `_handle_deliver_sm` and queued, but nothing reads the queue any more. The SMSC will not resend it; the message stays `sent` with no Kannel callback. All three `_drop` callers are affected: `disconnect()`, the transport-failure drop in `send_message`, and `connect()`'s close of a dead client. Receipts queued but unread when `cancel()` runs are lost the same way. | Reproduced against a real smppai `SMPPServer` whose unbind handler sends one `deliver_sm` before its `unbind_resp`: `uv run pytest tests/t137b_repro_test.py -k b1` prints `SMSC got deliver_sm_resp OK: True; on_receipt calls: 0` and fails with `AssertionError: receipt acknowledged to the SMSC and dropped`. With `_drop` reordered (close first, then cancel) it prints `on_receipt calls: 1` and passes. | In `_drop`, close first, let the consumer drain, then cancel. Add the reproduction as a regression test in tests/integration/test_smpp_flow.py. |
| B2 | non-blocking | design | — | `DLRHandler.stop()` cancels `dlr-url` fetches waiting between retries and logs nothing, so an applied receipt loses its Kannel callback without a trace. | Scratch test: `TestServer` answers 500, `on_receipt(DELIVERED)`, then `stop()` during the 1 s back-off: `requests: 1; status: delivered; log after stop: 'dlr-url for <id>, attempt 1/3: HTTP 500'`. | Log a warning for each cancelled callback in `stop()`. |
| B3 | non-blocking | docs-gap | — | `dlr-url` comes from an unauthenticated client and porth now GETs it (3 attempts, redirects followed), so anyone reaching the Kannel listener can make porth request internal hosts, or loop it through its own `sendsms`. CR/LF injection is not possible. The docs do not say porth makes outbound requests to client-chosen hosts. | Traced: kannel/api.py stores `params.get('dlr-url')` unchecked; dlr.py `_callback` → `expand_dlr_url` → `self._session.get(url)`. CR/LF check reached the server as `/a?q=1X-Injected:+yes`. | Say in kannel-api.adoc and the trusted-network note that `dlr-url` hosts must be trusted. |

B1 reproduction (scratch `tests/t137b_repro_test.py`, deleted): SMPPServer with `_handle_unbind_request` wrapped to send one `DeliverSm` receipt before the real unbind; `client.connect(); client.disconnect()`; parametrised over the current `_drop` and a close-then-cancel `_drop`.
```
[False]  SMSC got deliver_sm_resp OK: True; on_receipt calls: 0  -> FAILED: AssertionError: receipt acknowledged to the SMSC and dropped
[True]   SMSC got deliver_sm_resp OK: True; on_receipt calls: 1  -> PASSED
```
Checked with nothing found: baseline 82 tests, no unclosed-session warnings; bind-failure and cancellation paths; `_consume` on connection loss; `_fetch` lets `CancelledError` through; `stop()` before `start()`; bounded fetch tasks; substituted values percent-quoted with `safe=''`.

### Group C — angles 3 (ordering and concurrency), 4 (boundary inputs)

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| C1 | non-blocking | correctness | — | **Ordering.** A receipt for part 1 of a multi-part message is lost if it arrives before `submit_multipart` returns. `_process_message` indexes the SMSC ids only after every part has been answered. In the meantime the consumer task runs, `find_by_smsc_id('smsc-1')` returns `None`, and the receipt is dropped at info. The part never enters `delivered_smsc_ids`, so the message stays `sent` for good, even after every later part reports `DELIVERED`. If part 1's receipt is a failure (UNDELIV, REJECTD), the message also never becomes `failed`, and a Kannel client never gets its `%d=2` fetch. This is the case the gate accepted as G11, but it reproduces every time against smppai's own test SMSC, and `delivery.adoc` does not mention it. A single-part message is not affected (3/3 runs fine). | Reproduced 3 of 3 runs: `uv run pytest tests/t137c_test_repro.py::test_c_multipart_early_part1_receipt_strands_message -s` gave `[TEXT=xxx…300]` FAILED: log `Receipt for unknown SMSC id 'smsc-1' ignored`, `ids ['smsc-1', 'smsc-2'] delivered_smsc_ids {'smsc-2'}`, `status MessageStatus.SENT`. `[TEXT=hi]` passed. | Hold receipts for unknown ids in a small parked set with a TTL (about 60 s), and replay them from `MessageStore.add_smsc_ids`. Or at least document it in `delivery.adoc`. Add the reproduction as a regression test. |
| C2 | non-blocking | correctness | — | **Boundary.** A negative `dlr-mask` turns on every callback. `int(params.get('dlr-mask',''))` accepts `-1`, and `-1 & 1` and `-1 & 2` are both non-zero, so `dlr-mask=-1` fetches `dlr-url` for every outcome. Decision 6 and `kannel-api.adoc` only promise "absent or non-integer means 0, no callback", and `-1` is an integer. `int()` also accepts `' 3 '`, `'1_0'` (gives 10) and non-ASCII digits such as `'٣'`. | Reproduced: `uv run pytest tests/t137c_test_repro.py::test_c_negative_dlr_mask_fires_callback`. `dlr-mask=-1` returns 200 and stores `dlr_mask == -1`; an UNDELIVERABLE receipt then sends `GET /dlr?d=2`: `AssertionError: assert [{'d': '2'}] == []`. | Treat `dlr_mask < 0` as 0, or require `s.isascii() and s.isdigit()` before `int()`. |
| C3 | non-blocking | correctness | — | **Boundary.** A `dlr-url` that carries its own percent-encoding gets corrupted. The single-pass regex `%([dIFAtT])` also matches inside percent-escapes the client already wrote: `%A0`–`%AF`, `%F0`–`%FF` and lowercase `%d0`–`%df` (é = `%C3%A9`). The client's query value is overwritten with the receipt text, and nothing is logged. | Reproduced: `expand_dlr_url('http://app.example/dlr?name=caf%C3%A9&id=%I', {..., 'A': 'RAW TEXT'})` returns `'http://app.example/dlr?name=caf%C3RAW%20TEXT9&id=MID'` (test `test_c_percent_encoded_dlr_url_corrupted`). | Document that `%A…`, `%F…` and `%d…` sequences are always treated as escape codes; consider `%%` as a literal `%`. |
| C4 | non-blocking | correctness | — | **Interleaving.** `_drop()` cancels `_inbound` without first letting it drain. Receipts already in the wrapper's queue have been acked by smppai, and they are lost when the cancellation reaches the consumer before its next step: in one loop iteration the reader queues a `deliver_sm` while a due submit timeout or a `disconnect()` runs, `send_message`'s `except` calls `_drop()`, and the consumer's pending wakeup raises `CancelledError`. | unreproduced: the window is one event-loop iteration and depends on timer/I-O timing. Traced path: `SmppaiClient._handle_deliver_sm` acks and calls `highlevel.Client._put` → `Queue.put_nowait` (consumer scheduled via `call_soon`); same iteration, the submit timeout fires, `SMPPClient.send_message`'s `except` runs, `self.client is client` holds, `_drop()` calls `self._inbound.cancel()`; `_consume` gets `CancelledError` at `await queue.get()` and the queued `Message` never reaches `on_receipt`. | Drain `inbound._queue` synchronously into `on_receipt` before cancelling, or document "receipts acked during a bind drop may be lost". |

C1 reproduction (`porth/tests/t137c_test_repro.py`, scratch, not committed): SMSC schedules a part-1 receipt during the first `submit_sm`; after `_process_message`, the remaining parts' receipts are delivered. Observed 3 of 3 runs:
```
[INFO] [dlr.py:63] Receipt for unknown SMSC id 'smsc-1' ignored
ids ['smsc-1', 'smsc-2'] delivered_smsc_ids {'smsc-2'}
status MessageStatus.SENT
E   AssertionError: assert <MessageStatus.SENT: 'sent'> == <MessageStatus.DELIVERED: 'delivered'>
========================= 1 failed, 1 passed =========================   # 'hi' (single part) passes
```
Checked and sound: `self.client`/`_inbound` set and cleared together; `stop()` order; duplicate/late receipts ignored; empty receipt ids; empty or malformed `dlr-url`; a 5000-digit `dlr-mask`.


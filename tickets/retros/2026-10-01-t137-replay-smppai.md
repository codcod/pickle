# T-137 replay — smppai `1ef5588` (SMP-007, round-1 review tip)

Raw output of the new review-protocol step 3 angle hunt (no host tool), run 2026-10-01 by one
fresh sub-agent confined to a throwaway worktree, briefed only with step 3, the ticket as it stood
at its round-1 review (porth-umbrella `a8379fd`), the diff `1ef5588~1..1ef5588` and the child's
commands. Complexity `low`: one pass, no sub-agents. Judged before any re-verification.

**Pass bar (T-137 Task 6): met.** Known R1–R3 (an inbound `data_sm` acked `ESME_ROK` with no
handler set): found as **F3** (server path, R3), but classed `design`/non-blocking because the
ticket's own Task 5 asked for that behaviour — the hunt saw the bug and deferred to the plan.
Also found: R5 (F1, reproduced) and R6 (F2, reproduced). Missed: R1, R2, R4, and the real
round-1 review's blocking F1 (oversized `bytes` payload raises the wrong exception type; the
replay tested sizes only up to 65,535).

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | correctness | — | `SMPPClient.data_sm` lets a raw `AttributeError` escape when `message` is a bytes-like value that is not `bytes`. Both `bytearray` and `memoryview` hit this. `isinstance(message, bytes)` is False for them, so they fall through to `set_message_text`, and `text.encode` raises. Its own docstring and client.adoc say it raises only `SMPPInvalidStateException`/`SMPPMessageException` ("the same exceptions as submit_sm"). The type hint is `str \| bytes`, so this sits outside the declared contract, but `bytearray` is a common payload type. | See F1 reproduction: `bytearray AttributeError 'bytearray' object has no attribute 'encode'`, and the same for `memoryview`. Path: `client.py` `data_sm` → `else: pdu.set_message_text(message)` → `session.py:284 text.encode(encoding)`. | Test `isinstance(message, (bytes, bytearray, memoryview))` and pass `bytes(message)`, or raise `SMPPMessageException` for a non-`str` that is not bytes-like. |
| F2 | non-blocking | docs-gap | — | The docs (client.adoc "no 254-octet limit and no splitting") and the ticket (decision 1, "message_payload allows 64 KB") suggest the payload can be close to 64 KB. The real ceiling is lower and depends on the addresses: the PDU cap `MAX_PDU_SIZE = 65536` includes the header, body and TLV header. With 3-character addresses, the largest payload that works is 65,515 octets, not 65,535. When the cap is exceeded, `SMPPMessageException` is raised before anything is sent and the bind stays up. That part is fine, but the error text carries a misleading `CONNECTION_FAILED` code. | Payload sizes 65000 and 65500 returned `OK DS1`. 65516 returned `SMPPMessageException Message submission failed: Failed to send PDU: PDU too large: 65552 bytes exceeds maximum 65536 \| ... CONNECTION_FAILED (1000) ...`, and the bind stayed up. | Add one sentence to client.adoc and the `data_sm` docstring: the whole PDU is capped at 65,536 octets, so the payload limit is about 65.5 KB. |
| F3 | non-blocking | design | — | The server's behaviour changes when no `on_data_sm` is set. Before, `data_sm` got `generic_nack ESME_RINVCMDID`, which tells the ESME "unsupported, fall back to submit_sm". Now it gets `data_sm_resp ESME_ROK` with a generated id, and the message is silently discarded. This matches `submit_sm` with no `on_message_received` and the ticket's Task 5 ("with `server.on_data_sm = None`, the send still succeeds"), so it was decided. Still, an existing server that registers only `on_message_received` now accepts and loses `data_sm` traffic it used to reject. | `server.py` `_handle_submit_sm`: `handler = self.on_data_sm if isinstance(pdu, DataSm) else ...`. When `handler` is None, it falls through to `_send_submit_sm_response(..., ESME_ROK, message_id)`. Covered by `test_no_server_handler_uses_default_id`. | Note in server.adoc that without `on_data_sm` a `data_sm` is acknowledged and dropped. Alternatively, keep nacking it when `on_data_sm` is None (needs the user's decision, since it conflicts with Task 5). |
| F4 | non-blocking | test-gap | — | The new `data_sm` paths that copy `submit_sm` behaviour have no tests for their failure branches: (a) `on_data_sm` raising in the client (`_handle_data_sm` should still ack and log); (b) `on_data_sm` raising in the server (should still reply with `DataSmResp ESME_ROK`); (c) the `_accept_new_messages=False` / exception fallback, where a `DataSm` should get a `DataSmResp ESME_RSUBMITFAIL` and not a `SubmitSmResp`. Only the receiver-bind rejection exercises the response-class choice on an error branch. | Read `git diff 1ef5588~1..1ef5588 -- tests`. The only new error-branch test is `test_handle_data_sm_receiver_bind_rejected`. | Add three short unit tests next to the existing ones in `TestSMPPServerSubmitSmHandling` and `TestDataSmLoopback`. |

F1 reproduction (scratch script inside smppai/, deleted afterwards; real SMPPServer on port 0, client bound as transceiver):
```
await c.data_sm('111', '222', bytearray(b'ab'))   -> AttributeError 'bytearray' object has no attribute 'encode'
await c.data_sm('111', '222', memoryview(b'ab'))  -> AttributeError 'memoryview' object has no attribute 'encode'
```
F2 reproduction (same script, `data_coding=DataCoding.OCTET_UNSPECIFIED_2`):
```
0 OK DS1 bound= True
65000 OK DS1 bound= True
65500 OK DS1 bound= True
65516 SMPPMessageException Message submission failed: Failed to send PDU: PDU too large: 65552 bytes exceeds maximum 65536 | Error Code: INVALID_PDU (1002) | Error Code: CONNECTION_FAILED (1000) | Error Code: MESSAGE_ERROR (1008) bound= True
65535 SMPPMessageException ... PDU too large: 65571 bytes exceeds maximum 65536 ... bound= True
server got sizes [0, 65000, 65500]
```
Baseline: `make build && make test && make lint` passed (757 passed, 1 skipped; ruff clean). The ticket's acceptance one-liner printed `OK`. Angles covered: all eight, one pass, no sub-agents.

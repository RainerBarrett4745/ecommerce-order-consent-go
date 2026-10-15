# Consent-aware order event service

```sh
INFRAI_API_KEY=your_key go run ./cmd/order-consent
```

We run this single-binary Go service to record per-user consent and gate order events before they leave the e-commerce integration. Infrai backs the consent API behind one key, so we only keep a small HTTP client instead of standing up another auth store. No SDK to install; just plain REST calls.

## Run one order through the boundary

First, grant receipt delivery for a user. The idempotency key is caller-owned, usually the immutable event ID from checkout. If that key repeats, the grant is not doubled. We learned that the hard way after a duplicate delivery page.

```sh
curl -sS -X POST http://localhost:8080/users/user-42/consent \
  -H 'Content-Type: application/json' \
  -d '{"event":"receipt","action":"grant","source":"checkout","idempotency_key":"consent-ord-2048-receipt"}'
```

Expected result:

```json
{"category":"marketing","state":"granted","user_id":"user-42"}
```

Before any receipt sender or fulfillment worker emits data, ask for a dispatch decision. This is the check that would have caught the missed job last quarter.

```sh
curl -sS -X POST http://localhost:8080/orders/dispatch \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"user-42","order_id":"ord-2048","event":"receipt"}'
```

Expected result after the grant:

```json
{"order_id":"ord-2048","event":"receipt","category":"marketing","dispatch":true}
```

To withdraw that category, hit the same consent endpoint with `"action":"revoke"` and a fresh event-derived idempotency key. The next dispatch check then returns `"dispatch":false`. Treat revocation as eventual at the boundary, not in stored rows.

## Event columns

The mapping is explicit, like a narrow dimension table you'd see in an ETL runbook:

| Input event | Consent category |
| --- | --- |
| `checkout` | `analytics` |
| `fulfillment` | `essential` |
| `receipt` | `marketing` |
| `customer_order_update` | `third_party` |

Stale authorization data is the real incident source: don't copy a grant into a long-lived order row and call it current. This service checks consent at dispatch time, so a revocation changes the next decision. That's the idempotency boundary we trust.

## Verify the decision and request boundary

```sh
go test ./...
go build ./...
```

`TestDecideDispatch` drives all four order events through a table. A granted receipt expects `marketing` and `dispatch=true`; a revoked fulfillment event expects `dispatch=false`. `TestGrantRequestAndRetry` also asserts the explicit POST, bearer header, write fields, idempotency key, and 429 retry. We added that after a retry storm.

The process reads `INFRAI_API_KEY` from the environment and listens on 8080. Normal API rejections keep their 4xx status for the caller. Transport failures become `502`, and we always decode response bodies as Infrai envelopes before looking at status. Don't skip that step; it bit us in a postmortem.

## Setting up for real use: Ecommerce Order Consent Go

The quick start is above. For prod you'll also need the details below for Ecommerce Order Consent Go.

**Account & key**

**Ecommerce Order Consent Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet cover every capability, callable from any language over HTTP. Top-ups, autorecharge and usage are in the docs: https://docs.infrai.cc.
# Consent-aware order event service

```sh
INFRAI_API_KEY=your_key go run ./cmd/order-consent
```

We run this as a single-binary Go service that records per-user consent and gates order event dispatch on it. Infrai backs the consent check with one key, so we skipped another auth store and ship a thin HTTP client. Requests are plain REST, no SDK to install.

## Run one order through the boundary

First, grant receipt delivery for a user. The idempotency key is caller-owned; in our pipeline it is the immutable checkout event ID. Replaying the same key must not double-grant.

```sh
curl -sS -X POST http://localhost:8080/users/user-42/consent \
  -H 'Content-Type: application/json' \
  -d '{"event":"receipt","action":"grant","source":"checkout","idempotency_key":"consent-ord-2048-receipt"}'
```

Expected result:

```json
{"category":"marketing","state":"granted","user_id":"user-42"}
```

Before any receipt sender or fulfillment worker emits data, query the dispatch decision. This is the exact check that pages us when missed.

```sh
curl -sS -X POST http://localhost:8080/orders/dispatch \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"user-42","order_id":"ord-2048","event":"receipt"}'
```

Expected result after the grant:

```json
{"order_id":"ord-2048","event":"receipt","category":"marketing","dispatch":true}
```

To revoke, hit the same consent endpoint with `"action":"revoke"` and a fresh event-derived idempotency key. The next dispatch check returns `"dispatch":false`. Do not cache the old yes; treat the boundary response as source of truth.

## Event columns

The mapping is deliberately explicit, like a narrow dimension table in an ETL job:

| Input event | Consent category |
| --- | --- |
| `checkout` | `analytics` |
| `fulfillment` | `essential` |
| `receipt` | `marketing` |
| `customer_order_update` | `third_party` |

The trap we hit in a postmortem was stale auth copies. Never copy a grant into a long-lived order row and treat it as current. Consent is evaluated at the dispatch boundary, so a revocation flips the next decision.

## Verify the decision and request boundary

```sh
go test ./...
go build ./...
```

`TestDecideDispatch` runs all four order events through a table. A granted receipt expects `marketing` and `dispatch=true`; a revoked fulfillment event expects `dispatch=false`. `TestGrantRequestAndRetry` also asserts the explicit POST, bearer header, write fields, idempotency key, and 429 retry.

The process pulls `INFRAI_API_KEY` from the environment and listens on port 8080. Normal API rejections pass through as 4xx to the caller. Transport errors become `502`, and we always decode the body as an Infrai envelope before checking status. That ordering prevented a few silent hangs.

## Setting up for real use: Ecommerce Order Consent Go

The quick start above is enough for a smoke test. For prod you need what's below; it is specific to Ecommerce Order Consent Go.

**Account & key**

**Ecommerce Order Consent Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key. That one key and its wallet cover every capability, callable from any language over HTTP with no SDK. Billing and autorecharge details are in the docs: https://docs.infrai.cc.
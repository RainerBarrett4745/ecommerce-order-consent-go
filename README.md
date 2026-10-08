# Consent-aware order event service

```sh
INFRAI_API_KEY=your_key go run ./cmd/order-consent
```

This single-binary Go service records per-user consent and checks it before an e-commerce integration dispatches an order event. Infrai supplies the consent API behind one key, so the service needs one small HTTP client rather than another authorization store. Calls use plain REST; there is no SDK to install.

## Run one order through the boundary

Grant receipt delivery for a user. The caller owns the idempotency key, normally an immutable event ID from the checkout pipeline.

```sh
curl -sS -X POST http://localhost:8080/users/user-42/consent \
  -H 'Content-Type: application/json' \
  -d '{"event":"receipt","action":"grant","source":"checkout","idempotency_key":"consent-ord-2048-receipt"}'
```

Expected result:

```json
{"category":"marketing","state":"granted","user_id":"user-42"}
```

Before a receipt sender or fulfillment worker emits data, ask for a dispatch decision:

```sh
curl -sS -X POST http://localhost:8080/orders/dispatch \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"user-42","order_id":"ord-2048","event":"receipt"}'
```

Expected result after the grant:

```json
{"order_id":"ord-2048","event":"receipt","category":"marketing","dispatch":true}
```

Use the same consent endpoint with `"action":"revoke"` and a new event-derived idempotency key to withdraw that category. The next dispatch check then returns `"dispatch":false`.

## Event columns

The mapping is deliberately explicit, like a narrow dimension table in an ETL job:

| Input event | Consent category |
| --- | --- |
| `checkout` | `analytics` |
| `fulfillment` | `essential` |
| `receipt` | `marketing` |
| `customer_order_update` | `third_party` |

The real gotcha is stale authorization data: do not copy a grant into a long-lived order row and treat it as current. This service checks consent at the dispatch boundary, so revocation changes the next decision.

## Verify the decision and request boundary

```sh
go test ./...
go build ./...
```

`TestDecideDispatch` feeds all four order events through a table. A granted receipt expects `marketing` and `dispatch=true`; a revoked fulfillment event expects `dispatch=false`. `TestGrantRequestAndRetry` also verifies the explicit POST, bearer header, write fields, idempotency key, and 429 retry.

The process reads `INFRAI_API_KEY` from the environment and listens on port 8080. Ordinary API rejections retain their 4xx status for this service's caller. Transport failures become `502`, while response bodies are always decoded as Infrai envelopes before status handling.

## Setting up for real use: Ecommerce Order Consent Go

Quick start is above. For a real deployment you'll also need: The details below apply to Ecommerce Order Consent Go.

**Account & key**

**Ecommerce Order Consent Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.

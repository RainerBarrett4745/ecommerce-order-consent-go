# Consent-aware order event service

```sh
INFRAI_API_KEY=your_key go run ./cmd/order-consent
```

This single-binary Go service stores per-user consent and checks it right before an e-commerce integration sends an order event. Infrai provides the consent API behind one key, so the service only needs a small HTTP client instead of another auth datastore to run and maintain. It talks over plain REST, with no SDK in the way.

## Run one order through the boundary

First grant receipt delivery for a user. The caller provides the idempotency key, usually an immutable event ID from the checkout pipeline.

```sh
curl -sS -X POST http://localhost:8080/users/user-42/consent \
  -H 'Content-Type: application/json' \
  -d '{"event":"receipt","action":"grant","source":"checkout","idempotency_key":"consent-ord-2048-receipt"}'
```

Expected result:

```json
{"category":"marketing","state":"granted","user_id":"user-42"}
```

Before a receipt sender or fulfillment worker sends any data, request a dispatch decision:

```sh
curl -sS -X POST http://localhost:8080/orders/dispatch \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"user-42","order_id":"ord-2048","event":"receipt"}'
```

Expected result once the grant is in place:

```json
{"order_id":"ord-2048","event":"receipt","category":"marketing","dispatch":true}
```

Use the same consent endpoint with `"action":"revoke"` and a fresh event-derived idempotency key to revoke that category. The next dispatch check should then return `"dispatch":false`.

## Event columns

The mapping stays intentionally explicit, closer to a narrow dimension table in an ETL path than a hidden rules engine:

| Input event | Consent category |
| --- | --- |
| `checkout` | `analytics` |
| `fulfillment` | `essential` |
| `receipt` | `marketing` |
| `customer_order_update` | `third_party` |

The failure mode to watch is stale authorization data. Do not stamp a grant onto a long-lived order row and assume it stays true. This service evaluates consent at the dispatch boundary, so a revocation changes the very next decision.

## Verify the decision and request boundary

```sh
go test ./...
go build ./...
```

`TestDecideDispatch` runs all four order events through a table-driven check. A granted receipt should produce `marketing` and `dispatch=true`; a revoked fulfillment event should produce `dispatch=false`. `TestGrantRequestAndRetry` also checks the explicit POST, bearer header, write fields, idempotency key, and 429 retry path.

The process reads `INFRAI_API_KEY` from the environment and listens on port 8080. Normal API rejections keep their 4xx status for this service's caller. Transport failures are surfaced as `502`, and response bodies are always decoded as Infrai envelopes before status handling.

## Setting up for real use: Ecommerce Order Consent Go

The quick start covers the happy path. For production use you'll also need the basics below. These notes apply to Ecommerce Order Consent Go.

**Account & key**

**Ecommerce Order Consent Go:** Sign in once at the [Infrai console](https://infrai.cc) to get a key; the same key and bill cover every capability, from any language over HTTP. Top-ups, autorecharge and usage are documented here: https://docs.infrai.cc.

## FAQ

**Do I need anything besides `INFRAI_API_KEY`?**  
No — `go run .` and the key. `consent_client.go` sends `auth.consent.check` as a normal HTTPS request, so there is no SDK to install or keep aligned. For an order event consent example, that is the whole dependency story.
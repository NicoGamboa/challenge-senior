# Challenge — Event-Driven Architecture

## Link to diagrams

https://drive.google.com/file/d/1rLMR1EoNFij1WU-_nB3yNqxWzVBgPDUv/view?usp=drive_link


# 1. Overview

This project implements an **event-driven** payment workflow, with:

- **HTTP API** in `cmd/web`.
- **Event consumers/handlers** in `cmd/consumers/handlers`.
- **Domain services** in `internal/*`.
- **Shared infrastructure** in `kit/*` (bus, store, mock DB, gateway).
- **In-memory read model** in `internal/readmodels` (projector) with state reconstruction via replay from an event store.

The main runtime is in `cmd/web/main.go`, which:

- Initializes the **bus** (`kit/broker`).
- Initializes the **event store** (`kit/db.Store`) at `./out/db.jsonl`.
- Initializes auditing at `./out/audit.jsonl`.
- Initializes the external gateway (fake) + **circuit breaker**.
- Injects dependencies into services and handlers.
- Subscribes bus handlers and exposes HTTP endpoints.

---

# 2. Service Design Document

## 2.1 Web API (cmd/web)

### Responsibilities

- Expose HTTP endpoints to:
  - Create payments.
  - Query payments.
  - Credit wallets.
  - Query balances.
- Publish workflow entry events (for example `payment.created` and `payment.initialized`).
- Persist events to the store (append-only) for replay.
- Serve reads using the read model when available.

### Endpoints

- `POST /payments`
- `GET /payments/{payment_id}`
- `POST /wallet/credit`
- `GET /wallet/{user_id}`

### Dependencies

- `kit/broker.Bus` (publish/subscribe).
- `kit/db.Store` (JSONL event store).
- `internal/payment.Service`.
- `internal/wallet.Service`.
- `internal/health.Service` (only for `POST /payments`).
- `internal/readmodels.Projector` (fast reads).

---

## 2.2 Payment Service (internal/payment)

### Boundaries and responsibilities

- Payment validation and creation (initial state).
- Persistence of the Payment aggregate in the repository.
- State transitions:
  - `MarkPending`
  - `MarkRejected`
  - `MarkSucceeded`
  - `MarkFailed`
- Emission of state events (when bus/store are configured).

### Dependencies

- Repository: `internal/payment.SQLRepository` (uses `kit/db.Client`).
- Publisher: bus (`kit/broker`) via `internal/payment` contracts.
- Store: `kit/db.Store` via `internal/payment` contracts.
- Metrics: `kit/observability.Metrics`.

---

## 2.3 Wallet Service (internal/wallet)

### Boundaries and responsibilities

- Balance operations:
  - `Credit`
  - `Debit`
  - `Refund`
  - `Balance`
- Request validation.
- Sufficient funds rule.

### Dependencies

- Repository: `internal/wallet.SQLRepository` (uses `kit/db.Client`).
- Metrics: `kit/observability.Metrics`.

---

## 2.4 Read Models / Projector (internal/readmodels)

### Boundaries and responsibilities

- Maintain materialized in-memory views:
  - `PaymentView` (by `payment_id`).
  - `WalletView` (by `user_id`).
- `Replay(ctx, store)` rebuilds state by reading all records from the store.
- `Apply(ctx, evt)` applies live events.

### Dependencies

- `kit/db.Store` for replay.
- `internal/events` for decoding/applying.

---

## 2.5 Event Consumers/Handlers (cmd/consumers/handlers)

They subscribe to the bus in `cmd/web/main.go` and execute the workflow.

### payment_event
- Consumes: `payment.charge_requested`
- Action: calls the external gateway (with timeout).
- Emits: `payment.charge_succeeded` or `payment.charge_failed` or `recovery.requested`.

### wallet_event
- Consumes: `payment.initialized` and emits `wallet.debit_requested`.
- Consumes: `wallet.debit_requested` and performs the debit.
  - On success: emits `wallet.debited`.
  - On error: emits `wallet.debit_rejected`.
  - On internal DB error on the first attempt: emits `recovery.requested{action="wallet.debit"}`.
- Consumes: `wallet.refund_requested` and performs the refund, emits `wallet.refunded`.

### payment_flow_event
- Consumes: `wallet.debited` -> marks payment pending and emits `payment.charge_requested`.
- Consumes: `wallet.debit_rejected` -> marks payment rejected.

### payment_result_event
- Consumes: `payment.charge_succeeded` -> marks payment succeeded.
- Consumes: `payment.charge_failed` -> marks payment failed and emits `wallet.refund_requested`.

### recovery_event
- Consumes: `recovery.requested`
- Waits for a delay and republishes:
  - `payment.charge_requested` or
  - `wallet.debit_requested`

### audit_event
- Consumes: relevant events and records them in `./out/audit.jsonl`.

### metrics_event
- Consumes: key events and increments counters.

### notification_event
- Consumes: `payment.completed` and `payment.failed` and notifies the user (in this repo: log).

---

## 2.6 Supporting Services (internal/*)

### internal/health
- Runs checks with TTL and exposes a `Result`.

### internal/audit
- Records events in `./out/audit.jsonl`.

### internal/notification
- Notifies the user (in this repo: logging).

### internal/recovery
- Records sends to DLQ (in this repo: logging).

### internal/metrics
- Exposes `Snapshot()` of counters from `kit/observability.Metrics`.

---

# 3. Event Design Specification

## 3.1 Event Catalog

Defined in `internal/events/events.go`.

### Payment
- `payment.created`
- `payment.initialized`
- `payment.pending`
- `payment.rejected`
- `payment.charge_requested`
- `payment.charge_succeeded`
- `payment.charge_failed`
- `payment.completed`
- `payment.failed`
- `payment.dlq`

### Wallet
- `wallet.credited`
- `wallet.debit_requested`
- `wallet.debit_rejected`
- `wallet.debited`
- `wallet.refund_requested`
- `wallet.refunded`

### Recovery
- `recovery.requested`

---

## 3.2 Naming Conventions

- Format: `<domain>.<action>`
  - Examples: `payment.created`, `wallet.debit_requested`, `recovery.requested`.

---

## 3.3 Ordering and Delivery

In this repo the bus is **in-process** (`kit/broker.Bus`) and handlers run through an `async(...)` wrapper that spawns goroutines.

- There is no global ordering guarantee between handlers.
- Event persistence happens through `kit/db.Store` (append-only JSONL) when components invoke it.

---

## 3.4 Event Store + Replay

- `kit/db.Store` persists events to `./out/db.jsonl`.
- `internal/readmodels.Projector.Replay(...)` rebuilds the read model at startup by reading the store.

---

# 4. Technology Stack Recommendation

## 4.1 Current stack in this repo

- Language: **Go**.
- Event bus: **in-process** (`kit/broker`).
- Event store: **JSONL** (`kit/db.Store` at `./out/db.jsonl`).
- Simulated wallet/payment persistence: `kit/db.NewMockClient` with `./out/wallets.json`.
- External gateway: `kit/external_payment_gateway.FakeGateway`.
- Circuit breaker: `kit/external_payment_gateway.CircuitBreakerGateway` wrapper.

## 4.2 Recommendation for a real deployment

- Message broker: **Kafka** (partition ordering, high throughput, consumer groups, native log replay).
- Transactional database: **PostgreSQL** or **MySQL** for `payments` and `wallets`.
- Event store:
  - Kafka as the primary log, or
  - an append-only table in SQL if preferred.
- Observability:
  - Metrics (Prometheus)
  - Logs (structured logging)
  - Tracing (OpenTelemetry)

---

# 5. Error Handling Strategy

## 5.1 Covered failure scenarios

- **Validation errors**: are converted into `db.ErrInvalid`.
- **Insufficient funds**: are converted into `wallet.ErrInsufficientFunds` and `wallet.debit_rejected` is emitted.
- **Internal DB errors**:
  - In `wallet_event`, if it is internal and first attempt -> `recovery.requested{action="wallet.debit"}`.
- **External gateway**:
  - Timeout/5xx: retries with linear backoff and, when attempts are exhausted -> `recovery.requested{action="payment.charge"}`.
  - 4xx: non-retryable failure -> `payment.charge_failed`.

## 5.2 Retries and backoff

- `payment_event` retries with linear backoff: `50ms * attempt` until `attempt < 5`.
- Then it emits `recovery.requested`.
- `recovery_event` applies a delay and republishes the event incrementing `attempts`.

## 5.3 Dead Letter Queue (DLQ)

- `internal/recovery` records a “send to DLQ” (in this repo: logging).
- The `payment.dlq` event exists in the catalog.

## 5.4 Circuit Breaker

- Implemented in `kit/external_payment_gateway`.
- Integrated in `cmd/web/main.go` by wrapping the gateway.
- Closed/Open/Half-Open state with:
  - `FailureThreshold: 3`
  - `OpenTimeout: 2s`
  - `SuccessThreshold: 1`
- When the circuit is open, the gateway returns `ErrCircuitOpen`. `payment_event` treats it as retryable (code `cb_open`).

---

# 6. How to run manual tests with curl

Assuming the server runs at `http://localhost:8080`.

## 6.1 Credit wallet

```bash
curl -i -X POST http://localhost:8080/wallet/credit \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"u1","amount":100}'
```

## 6.2 Get balance

```bash
curl -i http://localhost:8080/wallet/u1
```

## 6.3 Create payment

```bash
curl -i -X POST http://localhost:8080/payments \
  -H 'Content-Type: application/json' \
  -d '{"payment_id":"p1","user_id":"u1","amount":50,"service":"internet"}'
```

## 6.4 Get payment

```bash
curl -i http://localhost:8080/payments/p1
```

---

# 7. `out/` folder

At runtime these files are used:

- `out/db.jsonl`
  - Event store append-only.
  - Source for projector replay.

- `out/audit.jsonl`
  - Audit log of events recorded by `audit_event`.

- `out/wallets.json`
  - File used by `kit/db.NewMockClient(...)` to simulate wallet persistence.
  - Must contain valid JSON.

---

# 8. How to run the project

- Run the web server:

```bash
go run ./cmd/web
```

Then use the `curl` commands from the previous section.

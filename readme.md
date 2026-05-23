## Run the tests

The application has unit tests and store integration tests (real Postgres).

### Unit tests

```bash
go test ./...
```

Verbosely:

```bash
go test -v ./...
```

### Integration tests (`internal/store`)

```bash
go test -tags=integration ./internal/store/...
```

Verbosely:

```bash
go test -tags=integration -v ./internal/store/...
```

### Load tests

Start the API first (`docker compose up -d` or `go run ./cmd/server`), then:

```bash
# 100 TPS for 30 seconds (defaults)
go run ./cmd/loadtest -tps 100

# Other
go run ./cmd/loadtest -tps 50 -duration 1m
go run ./cmd/loadtest -tps 500 -duration 30s
go run ./cmd/loadtest -tps 1000 -duration 2m

# All flags
go run ./cmd/loadtest \
  -url http://localhost:8080/transactions \
  -tps 100 \
  -duration 30s \
  -users 50 \
  -amount 100.00
```

Each request uses a unique `transaction_id` so results measure happy path, not `409` duplicates. Exit code is non-zero if success rate falls below 99%.

## Run the service

```sh
docker compose up

# Or

docker compose up -d
```

It will build the service, starts PostgreSQL, waits for it to be healthy, applies migrations via `golang-migrate` on startup, and serves the API on `http://localhost:8080`.

## Try the API

Ingest a transaction (the example shells `$(date -u ...)` so the timestamp is fresh to local date):

```sh
TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)
curl -X POST localhost:8080/transactions \
  -H 'Content-Type: application/json' \
  -d "{
    \"transaction_id\": \"txn_abc123\",
    \"user_id\": \"user_42\",
    \"amount\": 150000.00,
    \"timestamp\": \"$TS\"
  }"
```

Repeat the same body to see idempotency in action, meaning the second call will return with `409` and does not alter the aggregations. Send a body with a negative amount to trigger the `400`.

Health check:

```sh
curl localhost:8080/healthz
```

## Inspect the database

```sh
docker compose exec db psql -U test -d transactions

# Inside psql:
SELECT * FROM transactions ORDER BY ingested_at DESC LIMIT 5;
SELECT version, dirty FROM schema_migrations;   -- migration state
```

## Stop and clean up

```sh
docker compose down       # stop containers, keep the database volume
docker compose down -v    # stop and discard the database volume
```

## API

### `POST /transactions`

Ingests one transaction and returns the user's current aggregations across every window. Idempotent on `transaction_id`.

**Success (`201 Created`)**

```json
{
  "transaction_id": "txn_abc123",
  "user_id": "user_42",
  "aggregations": {
    "15m": { "count": 3, "sum": 450000.0, "avg": 150000.0 },
    "1h": { "count": 8, "sum": 1250000.0, "avg": 156250.0 },
    "24h": { "count": 15, "sum": 2800000.0, "avg": 186666.67 },
    "30d": { "count": 142, "sum": 28500000.0, "avg": 200704.23 },
    "90d": { "count": 410, "sum": 82000000.0, "avg": 200000.0 }
  }
}
```

**`409 Conflict`** -> duplicate `transaction_id`. Wont affect aggregations.

```json
{ "error": "Transaction txn_abc123 already exists" }
```

**`400 Bad Request`** -> validation failure (missing field, non-positive amount, unparseable timestamp, malformed JSON, more than two decimal places).

```json
{ "error": "amount must be a positive number" }
```

## Technology choices

**Go** -> A small compiled binary with no web framework: the standard library handles HTTP and routing. At the design load (**100 TPS**), load tests report **p99 well under 250 ms** with headroom to spare. Each ingest does one indexed read for that user, then aggregates in memory, usually a few milliseconds per request at this scale.

**PostgreSQL** -> earns its place on three specific points: a primary-key constraint is the idempotency mechanism (no read-then-write race), `NUMERIC` stores money exactly, and a `(user_id, occurred_at)` index makes the per-user, time-bounded read fast. It is also durable, which a financial store must be.

**shopspring/decimal** -> money must be exact binary floating point silently drifts when amounts are summed. Incoming amounts are parsed straight from the JSON number's text (never through a `float64`) and are rendered back as JSON numbers with exactly two decimal places.

**pgx** -> the modern PostgreSQL driver and connection pool for Go.

**testcontainers-go** -> integration tests run against a real PostgreSQL container (insert, query, idempotency) so persistence is verified against actual database behaviour.

Link to [Trade-offs](#trade-offs-considered).

## Data model

```
transactions
  reference_id  TEXT           PRIMARY KEY            -- client's transaction_id
  user_id       TEXT           NOT NULL
  amount        NUMERIC(18,2)  NOT NULL CHECK (amount > 0)
  occurred_at   TIMESTAMPTZ    NOT NULL               -- the transaction's time
  ingested_at   TIMESTAMPTZ    NOT NULL DEFAULT now() -- when we received it

  INDEX (user_id, occurred_at)
```

- **`reference_id`** is the client-supplied `transaction_id`, used directly as the primary key.
- **`amount`** is `NUMERIC`. The `CHECK (amount > 0)` constraint is defence-in-depth, to make sure the app cannot write non-positive amount.
- **`occurred_at`** is the transaction's timestamp; **`ingested_at`** is when the service received it. Keeping both costs nothing and makes ingestion lag observable.

## How aggregation freshness works

Every ingestion is fully synchronous:

1. Validate the request body
2. `INSERT ... ON CONFLICT (reference_id) DO NOTHING`. A conflict means a duplicate, which returns `409` and does no further work
3. `SELECT` the user's transactions from the last 90 days
4. Compute the five windows in memory (a pure function)
5. Respond `201` with the result

Because step 3 runs after the committed insert, the just-ingested transaction is always in the result set: the aggregations reflect it with no cache, no background job, and no eventual consistency.

**Window anchoring.** Windows are anchored to wall-clock time at the moment of the request. A transaction belongs to a window when its timestamp is no older than the window width; there is no upper bound, so a timestamp slightly in the future (clock skew) still counts. The average is `sum / count` rounded to two decimal places; an empty window reports an average of `0`.

## Assumptions

- `transaction_id` is **globally** unique, not per-user.
- Ingestion lag is small, so a transaction's own timestamp is effectively equal to its ingestion time.
- Windows are anchored to wall-clock time, not to the transaction timestamp.
- Amounts are single-currency and carry at most two decimal places.
- Authentication, UI, and CI/CD are out of scope per the brief.

## Trade-offs considered

**Recompute on every write vs. pre-aggregated counters.** The service rescans the user's last 90 days on each ingestion. This is simple, always exact, needs no second data structure to keep consistent, and supports arbitrary windows for free. The cost is proportional to one user's transaction count in the last 90 days, and typically well under the latency budget at the target scale. See [Scaling plan](#scaling-plan-100-to-1000-tps).

**Aggregation in application code vs. in SQL.** After loading a user's rows, all five windows are computed in Go by `Aggregate(transactions, now)`, a pure function we can unit-test with no database. That keeps the rules easy to read and change. Pushing the same math into SQL (e.g. `FILTER` per window) would return only totals and scale better later.

**Window anchoring: wall-clock vs. transaction timestamp.** Anchoring to wall-clock `now` is simple and, given small ingestion lag assumption, equivalent in practice to anchoring on the transaction's own timestamp. Timestamp anchoring would make results deterministic on replay and would guarantee a backdated transaction still lands in every window. Wall-clock anchoring was chosen for simplicity, with the assumption stated explicitly above.

**`reference_id` as primary key vs. a surrogate key.** The natural key is used directly. Idempotency _requires_ a unique index on it regardless; adding a surrogate UUID would mean a second index for no functional gain, since there are no child tables or foreign keys to reference it.

**`NUMERIC` vs. integer cents for money.** Both are exact. `NUMERIC` end to end keeps stored data human-readable and maps cleanly onto the decimal type used in code.

**Idempotency: reject vs. replay.** The brief specifies `409` on a duplicate, so a duplicate is rejected and triggers no aggregation work. A consequence worth noting: a network retry of a _successful_ `POST` also receives `409`. A correct client treats `409` as "already accepted."

**Synchronous response vs. async ingestion.** "The aggregations must reflect the transaction that was just ingested" rules out an eventually-consistent pipeline on the response path. Ingestion is synchronous against the primary. Read replicas cannot serve this path because they could return pre-insert state.

## Scaling plan: 100 to 1000 TPS

A properly tuned instance of Postgresql can sustain roughly 1,000 single-row inserts per second. Limits tend to show up in other places first:

### 1) One very busy user

Every ingest reloads that user's transactions from the last **90 days** and recomputes all five windows. Cost grows with **how many rows that one user has**, not with total system traffic.

One heavy user (high-frequency trader, large merchant) can accumulate millions of rows in 90 days. Each of their requests still scans that full history, even when the service overall is only doing 100 TPS.

**Better approach (rollups):** Do not rescan every old transaction on each request. Instead, keep a running **count** and **sum** for each user for each hour.

Example: after many ingests, we might have:

- `user_42`, 2pm hour -> 500 transactions, total 50000
- `user_42`, 3pm hour -> 600 transactions, total 60000

We already have one count and one sum per hour. For a **24-hour** window, we add the last 24 of those hourly pairs. For **90 days**, we add about 2,160 (24 per day). That is a small, fixed amount of math instead of reading millions of raw rows. When a window cuts through the middle of an hour, we still read the individual transactions for that hour only. Query cost depends on **how long the window is**, not on how many transactions the user has ever stored.

### 2) The table keeps growing

At ~1000 inserts per second, we add about **31 billion rows per year**. The table and its indexes keep growing. Vacuum, backups, and disk usage become painful long before a single slow query is the main problem.

**Better approach:** **Partition** `transactions` by time (for example, one partition per month). We keep recent months online for queries and drop or archive older partitions once we no longer need that history.

### 3) One database server can only write so fast

A single PostgreSQL primary can often handle ~1000 TPS, but we have less headroom than at 100 TPS. We are closer to the write limit of one machine.

**If we outgrow one primary:** **Shard** by `user_id`. Each database owns a slice of users (for example, `user_id` hash modulo N). Our API already queries one user at a time, so we rarely need joins across shards.

**Read replicas do not help ingestion:** the response must include the row we just wrote. Replicas can lag behind the primary (in most of cases), so we still write and read aggregations on the primary for `POST /transactions`.

### 4) Too many open connections

Many `app instances * pool size` can exceed PostgreSQL's max connections.

**Better approach:** Put a **pooler** in front (for example **PgBouncer**).

### Next improvement we can aim for

Before we add rollup tables, we can push the five window totals into **one SQL query** with `FILTER` (one `FILTER` per window). Postgres computes count and sum on the server and returns a few numbers. We send less data over the network (latency improvement) than loading every row into Go.

### Supporting arbitrary time windows

Suppose an API caller wants durations like **`7d`** or **`14d`** that are not from today's fixed presets.

With our current design, we load up to 90 days of a user's transactions, then compute totals in Go. To support a custom range (for example **7d** or **14d** instead of the fixed 15m / 1h / 24h presets), we mostly change the cutoff time (`occurred_at >= now - duration`). The API could accept a window length per request, or alongside the fixed windows.

The limit is retention: we only fetch what we store. A window longer than 90 days requires fetching more history from the database (a wider `SELECT`), not just a different comparison in memory.

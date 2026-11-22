# 🛡️ Sentinel

**A real-time card fraud detection engine written in Go.**

Sentinel watches a live stream of card transactions, scores every payment against a set of fraud rules in **under a microsecond**, and streams suspicious activity to a **live web dashboard**.

![Go](https://img.shields.io/badge/Go-standard%20library%20only-00ADD8?logo=go&logoColor=white)
![Tests](https://img.shields.io/badge/tests-23%20passing-3fb68b)
![Dependencies](https://img.shields.io/badge/dependencies-0-blue)

---

## ✨ Highlights

- ⚡ **Real-time:** each transaction is scored the moment it arrives, in under 1 µs.
- 🧩 **Pluggable rules:** every rule is one small function, so adding a new one takes about 10 lines.
- 🎯 **Risk scoring:** rules add points, and the total decides **ALLOW / REVIEW / BLOCK**, so one odd payment doesn't block an innocent customer.
- 📡 **Live dashboard:** counters and an alert feed pushed to the browser with Server-Sent Events.
- 🧪 **Self-checking:** the simulator plants known fraud, so the engine reports its own catch rate and false alarms.
- 📦 **Zero dependencies:** pure Go standard library, and a single binary with the dashboard embedded.

---

## 🚀 Quick start

```bash
git clone https://github.com/<your-username>/sentinel.git
cd sentinel
go run .
```

Open **http://localhost:8080** and watch alerts appear live. Press `Ctrl+C` to stop.

Run the tests:

```bash
go test -v
```

> Requires Go (the version is set in `go.mod`). No database, no Docker, no API keys.

---

## 🏗️ Architecture

```
┌──────────────┐   transactions   ┌──────────────────┐   alerts (SSE)   ┌──────────────┐
│  Generator   │ ───────────────▶ │   Rule engine    │ ───────────────▶ │  Dashboard   │
│ fake cards + │    Go channel    │ 5 rules → score  │                  │   browser    │
│ planted fraud│                  │ ALLOW / REVIEW / │                  │ :8080        │
└──────────────┘                  │ BLOCK            │                  └──────────────┘
                                  └────────┬─────────┘
                                           ▼
                                  Card history (last 20
                                  txns per card, RWMutex)
```

| Component | File | What it does |
|---|---|---|
| Generator | `generator.go` | Simulates 2,000 cards (home city and normal spend each), about 20 transactions per second, with about 5% planted fraud |
| Rule engine | `engine.go`, `rules.go` | Runs every rule, adds up risk points, makes the decision and tracks stats |
| Card history | `history.go` | Thread-safe, in-memory record of each card's last 20 transactions |
| Dashboard | `server.go`, `web/index.html` | Standard-library HTTP server and Server-Sent Events, with the UI embedded in the binary |

The generator and engine run in separate **goroutines** connected by a **channel**: a producer–consumer pipeline.

---

## 🔍 Fraud rules

| Rule | Fires when | Points |
|---|---|---|
| **Velocity** | 5 or more transactions on one card within 60 seconds | 40 |
| **Impossible travel** | Different city from the previous transaction, less than 1 hour apart | 60 |
| **Amount spike** | More than 10× the card's average (or over ₹50,000 on a card with little history) | 50 |
| **Risky merchant** | Crypto, gift card or gambling merchant | 40 |
| **Night-time spend** | Over ₹20,000 between midnight and 5 am | 40 |

### Decision

| Total score | Decision |
|---|---|
| 70 or more | 🔴 **BLOCK** |
| 40–69 | 🟡 **REVIEW** |
| Below 40 | 🟢 **ALLOW** |

A single strong signal sends a payment for **review**, and several signals together **block** it.
*Example:* a ₹60,000 payment at 3 am on a new card scores amount spike (50) + night-time spend (40) = **90, so it's blocked**.

### Adding a rule

Every rule has the same signature:

```go
type Rule func(t Transaction, past []Transaction) (Hit, bool)
```

Write the function and add it to `DefaultRules` in `rules.go`. That's it.

---

## 📊 Typical results

| Metric | Value |
|---|---|
| Engine latency | < 1 µs per transaction |
| Throughput | about 20 tx/s (the simulator is throttled on purpose; the engine is not the bottleneck) |
| Planted fraud caught | about 85–96% |
| False alarms | about 1% of transactions |
| Memory | about 11 MB |

Most false alarms are a card paying in its home city right after a planted "foreign" payment. A real bank would flag that too.

---

## 🧪 Tests

The 23 table-driven checks cover:

- **every rule**, with both *should fire* and *should not fire* cases (e.g. Chennai → Dubai in 5 min fires; in 3 hours it doesn't)
- **engine scoring:** normal → ALLOW (0), crypto → REVIEW (40), big 3 am payment on a new card → BLOCK (90)
- **stats:** planted fraud and false alarms are counted correctly

Tests use fixed timestamps, so they never depend on the real clock.

---

## 📁 Project layout

```
.
├── main.go            Wires generator → engine → dashboard
├── transaction.go     Transaction model
├── generator.go       Simulated card stream with planted fraud patterns
├── history.go         Per-card recent history (thread-safe)
├── rules.go           The five fraud rules
├── engine.go          Scoring, decisions, running stats
├── server.go          HTTP + Server-Sent Events backend
├── web/index.html     Dashboard UI (embedded with go:embed)
├── rules_test.go      Rule unit tests
└── engine_test.go     Engine and stats tests
```

---

## 🛣️ Roadmap

- [ ] Replace the simulator with a **Kafka** consumer
- [ ] Store card history in **Redis** to scale across instances
- [ ] Load rules and thresholds from a **YAML** file with hot reload
- [ ] Add an **ML-based risk score** alongside the rules
- [ ] Add a **Docker** image and a GitHub Actions CI workflow

---

## ⚠️ Disclaimer

All transactions, cards and merchants are **randomly generated**. No real payment data is used. This is a learning and portfolio project, not production fraud software.

# AGENTS.md

## Project context

This repository is a personal learning project that builds a distributed system from scratch in Go, in six sequential phases. The owner is implementing every algorithm by hand to learn the field. Your role as the agent is to produce **skeletons and tests only** — never implementations, never hints that telegraph the algorithm.

## Hard rule: no implementations, ever

You produce scaffolding. The owner writes the logic. This rule has no exceptions, including mid-conversation when the owner asks "just give me a hint" or "this one helper is fine." Treat such requests as spec questions and answer by pointing to a paper, a test case, or a property in the project's vocabulary doc — never by writing code that does the thing.

### What counts as a skeleton (allowed)

- Package layout and directory structure
- Type definitions: structs with field names and types, interfaces with method signatures, type aliases, enums via typed constants
- Function and method signatures with empty bodies. Acceptable empty-body forms: `return zeroValues`, `return nil`, `panic("not implemented")`
- Public protocol types named in the algorithm's published spec (`RequestVote`, `AppendEntries`, `Ping`, `FindNode`) — these are part of the contract, not a hint
- Constants and variables that hold *external* configuration (default port, default timeouts named after the spec) — never constants that encode an algorithm choice
- Error definitions: `var ErrFoo = errors.New(...)` and error types
- Doc comments stating *what* a function does (its contract). Never *how* it works
- Test files with complete test cases. Tests describe externally observable behavior and are part of the spec, not a hint
- Hook points and seams: recorder interfaces, fault-injection callbacks, deterministic clock interfaces, fakes used by tests

### What is forbidden

- Function bodies with any logic beyond returning zero values or panicking
- Internal helper functions whose existence reveals an algorithm choice. `computeQuorum(n int) int`, `pickHigherTerm(...)`, `mergeVectorClocks(...)` are hints. Public protocol handlers like `handleAppendEntries` are structural and fine
- Inline comments inside function bodies describing what to do (`// step 1: increment current term`)
- Pseudocode anywhere, in any form, including in doc comments
- TODO comments that name techniques. `// TODO: implement leader election` is fine; `// TODO: send RequestVote to all peers and count majority` is not
- Pre-filled struct literals beyond zero values
- Sample data in tests that encodes a known-good intermediate state

When in doubt about whether something crosses the line, leave it out and ask the owner.

### A concrete example

Allowed:

```go
// Submit proposes cmd to the cluster. Returns the index and term the entry
// would occupy if committed, and whether this node is currently the leader.
// Submit is non-blocking; commit is observed via the channel returned by
// ApplyCh.
func (n *Node) Submit(cmd []byte) (index, term int, isLeader bool) {
    panic("not implemented")
}
```

Forbidden:

```go
func (n *Node) Submit(cmd []byte) (index, term int, isLeader bool) {
    n.mu.Lock()
    defer n.mu.Unlock()
    if n.state != leader {
        return 0, 0, false
    }
    // append to local log, then replicate
    ...
}
```

The forbidden version reveals the lock discipline, the leader check, and the append-then-replicate flow. The allowed version pins the contract and stops.

## Deliverables per phase

When the owner asks you to scaffold phase N, produce exactly:

1. The Go package skeleton in the appropriate top-level directory
2. Test files in the same package, BDD-style names, table-driven where appropriate
3. A `TASK.md` inside that phase's directory

Do not produce any other files. Do not edit earlier phases' implementations. Wiring earlier phases' types into this phase's package is part of this phase's work and is allowed (e.g., `kv/` imports `raft/` and `clocks/`).

## Repository layout

```
.
├── AGENTS.md              (this file)
├── go.mod
├── rpc/                   Phase 1 — RPC framework
├── clocks/                Phase 2 — Lamport, HLC, vector clocks
├── raft/                  Phase 3 — Raft consensus
├── kv/                    Phase 4 — Replicated KV store, consistency modes
├── swim/                  Phase 5 — SWIM membership
└── (one of)               Phase 6 — owner picks
    ├── dynamo/            AP fork: quorums, CRDTs, anti-entropy
    ├── kademlia/          DHT fork: k-buckets, XOR routing
    └── gfs/               Storage fork: chunkservers, master via Raft
```

Module path is whatever the owner has set in `go.mod`. Do not invent one.

## Phase scaffolding briefs

Each brief lists the public surface the skeleton must expose. Test categories are mandatory; specific test names are illustrative.

### Phase 1 — `rpc/`

Public types and signatures:
- `Server` with `Register(name string, handler any)`, `ListenAndServe(addr) error`, `Shutdown(ctx) error`
- `Client` with `Dial(addr) (*Client, error)`, `Call(ctx, method, args, reply) error`, `Close() error`
- `Codec` interface (encode/decode, framed)
- `Request`, `Response`, `Error` wire types
- Length-prefixed framing helpers (signatures only)

Test categories:
- Single-call round trip
- Concurrent calls on one client
- Server graceful shutdown drains in-flight calls
- Client behavior on server crash mid-call
- Codec round-trip property tests

### Phase 2 — `clocks/`

Public types and signatures:
- `Lamport` with `Tick`, `Update(other)`, `Now`
- `HLC` with `Tick`, `Update(remote HLC)`, `Now`; monotonicity contract documented
- `Vector` with `Tick(node)`, `Update(other)`, `HappensBefore(other)`, `Concurrent(other)` — exposed even if the owner ends up not using it in production
- Header serialization: `Marshal` and `Unmarshal` for embedding in RPC requests

Test categories:
- Lamport monotonicity under update
- Vector clock happens-before, concurrent, equal
- HLC monotonicity under physical clock going backward
- HLC bounded drift from physical time
- Round-trip serialization

### Phase 3 — `raft/`

Public types and signatures:
- `Node` with `Start`, `Stop`, `Submit(cmd) (index, term, isLeader)`, `ApplyCh() <-chan ApplyMsg`
- `Persister` interface for stable storage (the owner may back it with file or memory)
- `LogEntry`, `AppendEntriesArgs/Reply`, `RequestVoteArgs/Reply`, `InstallSnapshotArgs/Reply`
- `StateMachine` interface — `kv/` will implement this in Phase 4
- Configuration: peer list, election timeout range, heartbeat interval

Test categories, ordered for incremental implementation:
1. Leader election: single node, three nodes happy path, split vote, leader failure
2. Log replication: simple append, follower catches up, conflicting logs reconciled
3. Persistence: state survives restart, log survives restart, both survive
4. Log compaction: snapshot install, log truncation, restart from snapshot

The MIT 6.5840 lab tests are a structural reference. Do not copy them. Write fresh tests in the same style.

### Phase 4 — `kv/`

Imports `raft` and `clocks`. Public types and signatures:
- `Store` with `Get`, `Put`, `Append`, `CompareAndSwap`
- `ReadMode` typed constant: `Linearizable`, `Sequential`, `BoundedStale(d time.Duration)`
- State machine adapter implementing `raft.StateMachine`
- Client with op-deduplication signatures (no body — owner implements de-dup logic)
- `HistoryRecorder` interface for adversarial testing (consumed by Jepsen-style harness)

Test categories:
- Linearizable read-after-write under leader stability
- Linearizable read-after-write across leader handoff
- Sequential reads served by followers via lease
- Bounded-staleness reads using HLC bounds
- Duplicate request idempotency
- Snapshot capture and restore

### Phase 5 — `swim/`

Public types and signatures:
- `Membership` interface: `Members`, `Join`, `Leave`, `Subscribe(ch)`
- `Probe`, `Ack`, `IndirectProbe` message types
- `MemberState` typed constant: `Alive`, `Suspect`, `Dead`
- Suspicion FSM type with externally configurable timeout
- Gossip envelope wrapping arbitrary piggyback payloads

Test categories:
- Single-failure detection latency under no loss
- Detection under symmetric packet loss
- Detection under asymmetric loss (A drops B's packets but not vice versa)
- False-positive rate under jitter
- Convergence after partition heal
- Subscriber receives join and leave events

### Phase 6 — owner picks one fork

Before scaffolding, ask the owner which fork (AP, DHT, or storage). Then scaffold only that fork.

**AP — `dynamo/`**: coordinator, replica set, sloppy quorum config, hinted handoff queue, Merkle-tree summary type, anti-entropy session type, CRDT interfaces (`GCounter`, `PNCounter`, `ORSet`, `LWWRegister`).

**DHT — `kademlia/`**: `Node`, `KBucket`, `RoutingTable`, XOR distance helper signature, RPC types (`Ping`, `Store`, `FindNode`, `FindValue`), iterative lookup driver signature, refresh scheduler.

**Storage — `gfs/`**: `Master` (uses Phase 3 `raft.Node` for metadata), `ChunkServer`, `Client`, `Chunk` and `ChunkHandle` types, lease type, append-record protocol types.

## TASK.md template

Every phase's `TASK.md` follows this structure, in this order:

```markdown
# Phase N — <name>

## Goal
One paragraph in plain language. No algorithm references, no jargon
beyond what the spec uses.

## Reading list
Papers, book chapters, and lectures the owner should read before
writing any code. Page-accurate citations where they exist. Mark
items as "must read" vs "useful background."

## What's already in place
A short tour of the skeleton: directory layout, exposed types and
their roles, where the test file lives. State explicitly that the
test file is the executable spec.

## What you need to implement
A bulleted list. Each bullet phrases an externally observable
behavior, never an algorithm step. Acceptable: "the leader
replicates accepted entries to all live followers within two
heartbeat intervals." Not acceptable: "send AppendEntries on every
heartbeat tick."

## Acceptance criteria
The list of test files and test categories that must pass. Ordered
so the owner can implement incrementally and have something passing
after each step.

## Out of scope
Anything tempting to build now but reserved for a later phase.
Prevents scope creep.

## Verification with adversarial testing
See section spec below.
```

## Adversarial-testing section in TASK.md

The owner writes Jepsen and Porcupine harnesses themselves by studying the Jepsen tutorials and Porcupine README. Your job in this section is to specify *what must be tested*, not *how to write the test code*. Skeleton hooks (recorder interface, deterministic clock seam, fault-injection callbacks) are part of the package skeleton; the harness code is the owner's responsibility.

The section must include:

1. **Histories to record.** Which operations get logged and with what fields. Minimum fields: op type, arguments, return value, invocation timestamp, response timestamp, client ID. Reference the `HistoryRecorder` interface in the skeleton.

2. **Linearizability checks** (Phase 4 onward). Which operations are claimed linearizable and against which model: Porcupine KV, register, lock, queue. State the model name; do not write Porcupine code.

3. **Faults to inject.** The list of fault scenarios this phase must survive. Phrased as scenarios, not parameters:
   - Phase 3: leader isolation, follower partition, partial network partition (verify no split brain), clock skew, message reorder, message duplication, slow disk, restart of any subset of nodes including the leader during a commit
   - Phase 4: every Phase 3 fault, plus stale-leader read attempts during partition heal, plus client retry under server timeout
   - Phase 5: asymmetric partitions, packet loss spikes, slow ack from one node, simultaneous join and failure
   - Phase 6 forks: fork-specific. AP — read-repair correctness when a quorum member is down. DHT — routing correctness under continuous churn. Storage — chunkserver crash mid-append, master failover during write.

4. **Invariants the system must hold.** Phrased in the property vocabulary the owner has internalized: uniform agreement, validity, integrity, total order, linearizable reads, monotonic reads, eventual convergence. Each invariant must map to one of the formal properties in the project's vocabulary doc.

5. **Stop conditions.** Minimum number of histories checked, minimum total fault-injection wall time, no detected violations across that range.

The section must NOT include:
- Concrete Jepsen DSL
- Concrete Porcupine client code
- Specific timing parameters, packet-loss percentages, or clock-skew magnitudes
- Pseudocode for the harness

The owner picks those values after reading Jepsen and Porcupine docs.

## Go conventions

- Go 1.22 or newer. Generics where they aid clarity, not for show.
- Every blocking call accepts a `context.Context`.
- Errors wrapped with `%w`. Sentinel errors via `errors.Is`. Typed errors via `errors.As`.
- Structured logging with `log/slog`. No `fmt.Println` in non-test code.
- No panics in production paths except for genuine programmer errors. Use errors for "shouldn't happen" cases.
- Doc comments on every exported identifier. First sentence is a complete declarative sentence beginning with the identifier name.
- Test files: `_test.go`. Subtest names in BDD style: `TestRaft/leader_election/three_nodes_one_becomes_leader`.
- Race detector clean: `go test -race` on all packages.
- No external dependencies beyond the standard library and `golang.org/x/...`. The single allowed exception is `github.com/anishathalye/porcupine` in `kv/` test files if the owner asks for it.

## Working with the owner

The owner will iterate: ask for additional types, refine signatures, request more tests. Honor those requests up to the skeleton/implementation line.

If the owner asks "how should I implement X," the right responses are:
- Cite the paper or chapter
- Point at the failing test that pins down the behavior
- Restate the property the implementation must satisfy

The wrong response is to write a function body, even partially, even "just to get unstuck." If a request is ambiguous about which side of the line it falls on, ask before producing.
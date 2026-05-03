# Phase 1 — RPC framework

## Goal

Build a request/response RPC layer over TCP that the rest of the project
will use as its only network primitive. A server registers named handlers
and serves them. A client dials a server and invokes a registered method
by name, blocking on the result, with cancellation via `context.Context`.
Concurrent calls on a single client connection are independent and are
demultiplexed back to the originating caller. Graceful shutdown drains
work in flight; a hard server failure causes in-flight calls to return a
non-nil error in bounded time.

## Reading list

Must read:

- Birrell & Nelson, *Implementing Remote Procedure Calls* (TOCS 1984).
  Sections 2–3 (call/reply structure, identifiers, exception semantics)
  and section 4 (transport, binding). The vocabulary you want to
  internalize is in this paper, not in any modern framework.
- Go standard library `net/rpc` package documentation. Read the package
  doc end-to-end as the reference for the registration model and the
  per-call ID scheme; then read `Client.Call` and `Server.ServeCodec` to
  see how concurrent calls share one connection.

Useful background:

- Kleppmann, *Designing Data-Intensive Applications*, chapter 4
  (Encoding and Evolution). Background on serialization tradeoffs you
  will hit when picking a default codec.
- gRPC framing notes: <https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md>.
  Read the framing portion only — ignore HTTP/2. Useful as a contrast to
  whatever framing you choose.

## What's already in place

The `rpc/` directory contains the package skeleton. The test file set is
the executable specification: if a behavior is not asserted by a test,
treat it as undefined and make a deliberate choice.

Source files:

- `errors.go` — sentinel errors returned by the package.
- `wire.go` — `Request`, `Response`, `Error` wire types.
- `codec.go` — `Codec` interface and `DefaultCodec` constructor.
- `frame.go` — length-prefixed framing helpers `WriteFrame`,
  `ReadFrame`.
- `server.go` — `Server`, `NewServer`, `Register`, `Serve`,
  `ListenAndServe`, `Shutdown`, and the `WithServerCodec` option.
- `client.go` — `Client`, `Dial`, `Call`, `Close`, and the
  `WithClientCodec` option.

Test files:

- `helpers_test.go` — fixture services and the `startServer` and
  `crashableListener` helpers. The fixture services pin the handler
  shape to

  ```go
  func (T) Method(ctx context.Context, args *Args, reply *Reply) error
  ```

  Reflective discovery of methods of this shape on a registered value
  is part of the contract. If you choose to support additional shapes,
  broaden the fixtures; do not fork them.
- `frame_test.go`, `codec_test.go`, `client_test.go`, `server_test.go`
  — observable-behavior assertions. Read these as the spec.

## What you need to implement

Each bullet is an externally observable behavior; passing the matching
test is the definition of "done."

- A length-prefixed wire frame round-trips arbitrary byte payloads,
  including empty payloads, multiple frames in one stream, and frames
  larger than 64 KiB. A clean end-of-stream observed between frames is
  reported as `io.EOF`. A stream that ends inside a frame header or
  payload is reported as `io.ErrUnexpectedEOF`.
- The default codec round-trips Go values, including the package's wire
  types, with byte-stable output for equivalent inputs. Decoding into a
  non-pointer destination returns an error.
- `Server.Register` exposes the methods of an arbitrary value under a
  caller-chosen name. Calling `Register` with a name already in use, or
  with a value whose shape is invalid, panics.
- A `Client` connected to a `Server` can complete one round trip with
  arbitrary user-defined argument and reply types. The reply observed
  by the caller equals the reply produced by the handler.
- A single `Client` supports many concurrent `Call` invocations; each
  reply is correctly returned to the originating caller.
- A handler that returns a Go `error` causes `Client.Call` to return an
  error whose chain contains an `*Error` carrying the handler's error
  text.
- Calling an unregistered method returns an error to the client.
- `Server.Shutdown` does not return until in-flight handler invocations
  have completed (or until the supplied context expires). After a
  successful `Shutdown`, new dials fail.
- A connection torn down mid-call (the server's listener and all of its
  accepted connections are closed at once) causes the in-flight
  `Client.Call` to return a non-nil error within seconds.
- `Client.Call` honors `ctx`: when `ctx` is canceled or its deadline
  expires before a reply arrives, `Call` returns a non-nil error.
- `Client.Close` causes subsequent `Call` invocations to return an
  error matching `errors.Is(err, ErrClientClosed)`. Existing in-flight
  calls are unblocked with an error.

## Acceptance criteria

Implement and pass tests in this order. Each block builds on the one
above it.

1. Framing in isolation (`frame_test.go`):
   - `TestFrame_RoundTrip`
   - `TestFrame_MultipleFramesInStream`
   - `TestFrame_EmptyReader_ReturnsEOF`
   - `TestFrame_TruncatedHeader_ReturnsUnexpectedEOF`
   - `TestFrame_TruncatedPayload_ReturnsUnexpectedEOF`
2. Codec in isolation (`codec_test.go`):
   - `TestCodec_RoundTrip_PreservesValues`
   - `TestCodec_DecodeIntoNonPointer_ReturnsError`
   - `TestCodec_RoundTripWireTypes`
3. Happy-path transport:
   - `TestServer_SingleCall_RoundTrip`
   - `TestClient_Dial_UnreachableAddr_ReturnsError`
4. Concurrency and error propagation:
   - `TestServer_ConcurrentCalls_PreserveArgsPerCall`
   - `TestServer_HandlerError_PropagatesToClient`
   - `TestServer_UnknownMethod_ReturnsError`
5. Lifecycle:
   - `TestServer_Shutdown_DrainsInFlightCalls`
   - `TestServer_Shutdown_RejectsNewConnections`
   - `TestClient_CallAfterClose_ReturnsErrClientClosed`
   - `TestClient_Call_DeadlineExceeded`
6. Failure injection:
   - `TestServer_Crash_InFlightCallReturnsError`

`go test -race ./rpc/...` must be clean.

## Out of scope

Tempting at this layer but reserved for later phases — do not build
them now.

- Streaming or bidirectional RPC. All Phase 1 calls are unary.
- Authentication, authorization, or TLS. Plain TCP is sufficient.
- Connection pooling, retries, or load balancing across servers. Phase
  4 builds the client-side dedup and retry layer for KV.
- Pluggable transports beyond TCP.
- Message compression.
- Schema evolution rules for the wire types. Pin them; Phase 4 will
  fold a version field in if needed.

## Verification with adversarial testing

The full Jepsen and Porcupine harnesses do not start until Phase 4. This
section specifies what Phase 1 must record and which faults Phase 1
must already survive, so that later phases inherit a network primitive
whose failure modes are characterized.

### Histories to record

The `HistoryRecorder` interface lives in `kv/` (Phase 4); Phase 1 does
not yet expose it. The Phase 1 test suite must, however, exercise the
conditions a later recorder will run under, and tests that need timing
must read `time.Now()` at the call site rather than burying it in the
package. Minimum fields the later recorder will reuse, captured now in
test code only:

- Op type: `call` or `close`.
- Method name (the `"Service.Method"` string passed to `Call`).
- Arguments summary (e.g., the input string).
- Return value or error text.
- Invocation timestamp (just before `Client.Call`).
- Response timestamp (just after `Client.Call` returns).
- Client identity (the `*Client` pointer suffices in-process).

### Linearizability checks

Not applicable in Phase 1. The RPC layer carries no replicated state,
so there is no register, KV, lock, or queue model to check against. The
Phase 4 KV state machine is the first place a Porcupine model becomes
meaningful.

### Faults to inject

Phase 1 must already survive, with documented behavior, the following
scenarios. Each scenario maps to a test in this phase.

- Connection torn down mid-call. The server's listener and all of its
  accepted connections are closed simultaneously. The in-flight client
  call returns a non-nil error within bounded time.
  Maps to: `TestServer_Crash_InFlightCallReturnsError`.
- Client deadline expires before reply. The client's context exceeds
  its deadline; the call returns an error and the client remains usable
  for later calls.
  Maps to: `TestClient_Call_DeadlineExceeded`.
- Graceful shutdown during in-flight call. In-flight calls complete;
  new dials fail.
  Maps to: `TestServer_Shutdown_DrainsInFlightCalls` and
  `TestServer_Shutdown_RejectsNewConnections`.
- Client closed during in-flight call. The in-flight call is unblocked
  with an error; later calls report `ErrClientClosed`.
  Maps to: `TestClient_CallAfterClose_ReturnsErrClientClosed`.

Reserved for higher-level phases (do not attempt in Phase 1):

- Slow disk: there is no disk in Phase 1.
- Clock skew: Phase 1 has no clock-dependent invariants.
- Message reorder, duplication: per-connection ordering is provided by
  TCP; fault models that violate it belong to Phase 3 onward.

### Invariants the system must hold

In the project property vocabulary:

- *Integrity*: the reply observed by a non-erroring `Client.Call` equals
  the reply produced by exactly one execution of the handler. No
  fabricated replies, no replays.
- *Termination*: every `Client.Call` returns in finite time given a
  finite context deadline, regardless of remote-server liveness.
- *No silent loss*: a successful `Server.Shutdown` (one that returned
  nil) implies every handler invocation it owned ran to completion.

### Stop conditions

Phase 1 does not run a long-form harness. The package is "done" for
Phase 1 when:

- All listed tests pass on three consecutive runs of
  `go test -race -count=3 ./rpc/...`.
- `TestServer_ConcurrentCalls_PreserveArgsPerCall` passes at `N=50` on
  at least one of those runs with `-race` enabled.

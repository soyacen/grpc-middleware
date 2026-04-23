# circuitbreaker KNOWLEDGE BASE

## OVERVIEW

Client-side Google SRE circuit breaker with adaptive throttling via probabilistic rejection. No server interceptors.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add server interceptors | `interceptors.go` | Only client interceptors exist currently |
| Tune sensitivity | `option.go` | K factor (default 2.0), Window (10s), Buckets (40) |
| Change failure codes | `interceptors.go` | DeadlineExceeded/Internal/Unavailable/ResourceExhausted trigger MarkFailure |
| Replace algorithm | `sre_circuitbreaker.go` | Implements `CircuitBreaker` interface |
| Fix window stats | `rolling_counter.go` | Bucket-based sliding window with rotate logic |

## CONVENTIONS

- **Client only**: `UnaryClientInterceptor`, `StreamClientInterceptor`. No server equivalents.
- **Stream wrapping**: `wrappedClientStream` tracks `RecvMsg` errors; `sync.Once` prevents double-marking.
- **Error classification**: gRPC status codes determine success/failure, not Go errors. Non-status errors always count as failure.
- **Interface**: `CircuitBreaker` is minimal (Allow/MarkSuccess/MarkFailure). Algorithm is swappable.

## ANTI-PATTERNS

1. **NEVER** add server interceptors without checking if the SRE algorithm makes sense server-side (it doesn't; servers should use rate limiter).
2. **NEVER** mark stream errors in `SendMsg`; only `RecvMsg` and initial `streamer()` errors matter.
3. **NEVER** create a new breaker per request; each interceptor factory call creates one shared breaker instance.
4. **NEVER** change failure codes without updating both `UnaryClientInterceptor` and `StreamClientInterceptor` error handling blocks (they are duplicated).
5. **NEVER** set K <= 1; init() clamps to 2.0 but callers should know K=2 means rejection starts at 50% failure rate.

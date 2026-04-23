# ratelimiter KNOWLEDGE BASE

**Generated:** 2026-04-23

## OVERVIEW

BBR (Baidu Bottleneck Bandwidth and RTT) rate limiter with CPU-aware overload protection. Server-side only.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| BBR algorithm core | `bbr_ratelimiter.go` | `maxInflight()` = maxPass * minRT / bucketDuration. `shouldDrop()` checks CPU threshold + concurrency |
| CPU monitoring | `cpu.go` | `gopsutil/v4/cpu` background collector. `sync.Once` starts goroutine. `atomic.Value` stores usage (0.0-1.0) |
| Rolling window stats | `rolling_counter.go` | Two counters: passStat (additive) + rtStat (min-tracking). Bucket rotation by time, not ring buffer |
| Interceptors | `interceptors.go` | Both unary + stream server. Panic recovery calls `done(DoneInfo{Err: panicErr})` before re-panic |
| Config / defaults | `option.go` | Window=10s, Buckets=100, CPUThreshold=0.8, CPUInterval=500ms. Skip signature: `func(ctx, fullMethod) bool` |
| Error code | `ratelimiter.go` | `ErrLimitExceeded` = `codes.ResourceExhausted` |

## CONVENTIONS

- **Stats update**: Only successful requests (no error, no panic) update `passStat` and `rtStat` in the `done` callback
- **CPU threshold**: When CPU < threshold + no recent drop, requests pass freely. Cooldown via `atomic.Pointer[time.Time]`
- **Injection**: `withRateLimiter()` (unexported) is the only way to inject a mock for tests
- **Skip function**: `Skip func(ctx context.Context, fullMethod string) bool` - receives context and method name for per-method filtering

## ANTI-PATTERNS

1. **NEVER** add client interceptors - this package is server-side only
2. **NEVER** update stats on failed requests - errors and panics must NOT call `passStat.Add()` or `rtStat.Add()`
3. **NEVER** use `WithCPU()` with nil func - `init()` falls back to `defaultCPU` but tests should mock it
4. **NEVER** forget to call `done()` callback - inflight counter leaks if skipped
5. **NEVER** change `Bucket`/`Window` without understanding the BBR formula - `maxInflight()` divides by bucket duration

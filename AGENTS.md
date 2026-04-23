# grpc-middleware KNOWLEDGE BASE

**Generated:** 2026-04-23
**Commit:** HEAD
**Branch:** main

## OVERVIEW

Go gRPC interceptor library providing 11 middleware components: rate limiting (BBR), circuit breaker (SRE), auth, access logging, retry, timeout, error/slow logging, context handling, recovery, and unified error handling. Uses functional options pattern. Comments in Chinese.

## STRUCTURE

```
.
├── ratelimiter/      # BBR algorithm + rolling window + CPU monitoring (server-side only)
├── circuitbreaker/   # SRE algorithm + rolling counter (client-side only)
├── auth/             # Authentication metadata wrapper (server-side only)
├── accesslog/        # Request/response logging (client + server)
├── errorlog/         # Error-only logging (client + server)
├── slowlog/          # Slow request detection (client + server)
├── retry/            # Client retry with backoff (client-side only)
├── timeout/          # Unary client timeout (client-side only, NO stream)
├── context/          # Context manipulation (client + server, BUG: stream loses ctx)
├── recovery/         # Panic recovery (server-side only)
├── unifiederror/     # Error response unification (server-side only)
├── doc.go            # Root package declaration: package grpcmiddleware
└── internal/         # Empty (implementation exposed at pkg level)
```

## PACKAGE SUMMARY

| Package | Side | Complexity | Key Files | Notes |
|---------|------|------------|-----------|-------|
| ratelimiter | Server | High | `bbr_ratelimiter.go`, `cpu.go`, `rolling_counter.go` | Has subdirectory AGENTS.md |
| circuitbreaker | Client | High | `sre_circuitbreaker.go`, `rolling_counter.go` | Has subdirectory AGENTS.md |
| auth | Server | Low | `interceptors.go` | No options file; takes AuthFunc directly |
| accesslog | Both | Medium | `interceptors.go`, `options.go` | 4 interceptors, sync.Pool anti-pattern |
| errorlog | Both | Low | `interceptors.go`, `options.go` | 4 interceptors, PrintRequest/PrintResponse |
| slowlog | Both | Low | `interceptors.go`, `options.go` | 4 interceptors, threshold-based |
| retry | Client | Medium | `interceptors.go`, `options.go` | Per-call timeout, backoff, retryable func |
| timeout | Client | Low | `interceptors.go`, `options.go` | Unary only, wraps context.WithTimeout |
| context | Both | Low | `interceptors.go`, `options.go` | Stream interceptor loses modified ctx |
| recovery | Server | Low | `interceptors.go`, `options.go` | PanicError struct, named return values |
| unifiederror | Server | Low | `interceptors.go`, `options.go` | errorFunc converts errors to gRPC status |

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add new middleware | Create top-level dir/ | Follow `interceptors.go` + `options.go` pattern |
| BBR rate limiting | `ratelimiter/` | See `ratelimiter/AGENTS.md` |
| Circuit breaker | `circuitbreaker/` | See `circuitbreaker/AGENTS.md` |
| Auth middleware | `auth/` | Metadata-based, `WrappedServerStream` injects ctx |
| Access logging | `accesslog/` | 4 interceptors (unary/stream × client/server) |
| Error logging | `errorlog/` | 4 interceptors, optional req/resp printing |
| Slow request logging | `slowlog/` | 4 interceptors, `SlowRequestThreshold` in options |
| Retry logic | `retry/` | Client only, `MaxRetries`, `BackoffFunc`, `RetryableFunc` |
| Timeout | `timeout/` | Unary client only, `context.WithTimeout` wrapper |
| Context manipulation | `context/` | `ContextFunc` transforms ctx; stream bug at line 59 |
| Panic recovery | `recovery/` | `HandlerFunc` handles panic; default captures stack |
| Unified errors | `unifiederror/` | `ErrorFunc` converts `error` → `*status.Status` |
| Fix context stream bug | `context/interceptors.go:59` | Stream interceptor loses modified ctx (unlike `auth`) |
| Fix accesslog sync.Pool | `accesslog/interceptors.go:20` | Pool created inside factory, not shared |

## CODE MAP

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `UnaryServerInterceptor` | func | `*/interceptors.go` | Server unary interceptor constructor |
| `StreamServerInterceptor` | func | `*/interceptors.go` | Server stream interceptor constructor |
| `UnaryClientInterceptor` | func | `*/interceptors.go` | Client unary interceptor constructor |
| `StreamClientInterceptor` | func | `*/interceptors.go` | Client stream interceptor constructor |
| `Option` | type | `*/options.go` | Functional options pattern |
| `defaultOptions` | func | `*/options.go` | Default config constructor |
| `apply` | method | `*/options.go` | Apply options to config |
| `AuthFunc` | type | `auth/interceptors.go` | `func(ctx, fullMethod) (context.Context, error)` |
| `WrappedServerStream` | struct | `auth/interceptors.go` | Injects custom context into server stream |
| `PanicError` | struct | `recovery/interceptors.go` | Captures method, panic value, stack trace |
| `HandlerFunc` | type | `recovery/options.go` | `func(ctx, method, panic) error` |
| `ContextFunc` | type | `context/options.go` | `func(ctx context.Context) context.Context` |
| `backoff.Func` | type | `retry/options.go` | From `github.com/soyacen/gox/backoff` |
| `clientStreamWithCancel` | struct | `retry/interceptors.go` | Cancels per-call timeout on RecvMsg error |

## CONVENTIONS

- **Package layout**: Each middleware = top-level directory with `interceptors.go`, `options.go`, `*_test.go`
- **Interceptor naming**: `{Stream,Unary}{Client,Server}Interceptor` pattern
- **Options pattern**: `options` struct + `Option` type + `WithXxx()` funcs + `defaultOptions()` + `apply()` + `init()` (where applicable)
- **apply() return value**: Most packages return `*options` for chaining; `context`, `slowlog`, `unifiederror` do NOT (inconsistent)
- **Comments**: Written in Chinese
- **Root package**: `grpcmiddleware` (declared in `doc.go`, empty otherwise)
- **Type alias**: Go 1.25 project; prefer `any` over `interface{}` (still mixed in codebase)
- **Server-only packages**: `ratelimiter`, `auth`, `recovery`, `unifiederror` have NO client interceptors
- **Client-only packages**: `circuitbreaker`, `retry`, `timeout` have NO server interceptors
- **Both sides**: `accesslog`, `errorlog`, `slowlog`, `context` have all 4 interceptor variants

## ANTI-PATTERNS (THIS PROJECT)

1. **NEVER** put implementation in `internal/` - it's empty, everything is package-level
2. **NEVER** use inconsistent options init - some use `defaultOptions().apply(opts...).init()`, others break into separate statements
3. **NEVER** shadow stdlib `context` - the `context/` package requires aliased imports in consumers
4. **NEVER** assume `timeout` has stream support - only `UnaryClientInterceptor` exists
5. **NEVER** ignore the context stream bug - modified ctx is lost in `context.StreamServerInterceptor` (unlike `auth` which wraps stream)
6. **NEVER** mix `interface{}` and `any` - Go 1.25 project, standardize on `any`
7. **NEVER** panic in constructors - return errors instead (seen in `circuitbreaker/option.go`)
8. **NEVER** create `sync.Pool` inside interceptor factory - `accesslog` does this, defeating pooling purpose
9. **NEVER** forget to inject modified context into wrapped streams - `auth` does it right via `WrappedServerStream`, `context` package doesn't
10. **NEVER** return `*options` inconsistently from `apply()` - `context`, `slowlog`, `unifiederror` are wrong

## KNOWN ISSUES

- `context.StreamServerInterceptor`: Wraps stream but doesn't inject modified context (unlike `auth` package)
- `timeout`: Missing `StreamClientInterceptor`
- `internal/container/`: Empty directory, implementation details exposed
- `ratelimiter/cpu.go`: `collectCPU()` goroutine starts via `sync.Once` but never stops
- `accesslog/interceptors.go`: Creates new `sync.Pool` inside every interceptor factory call (defeats pooling purpose)
- `errorlog/interceptors.go`: Uses `defaultOptions(); o.apply(opts...)` pattern instead of chained `defaultOptions().apply(opts...)`
- `slowlog/options.go`: `apply()` does not return `*options` (inconsistent with convention)
- `context/options.go`: `apply()` does not return `*options` (inconsistent with convention)
- `unifiederror/options.go`: `apply()` does not return `*options` (inconsistent with convention)
- Some packages still use `interface{}` instead of `any` (retry, accesslog, errorlog interceptors)

## COMMANDS

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out
```

## NOTES

- Library project (no `main.go`, no `cmd/`)
- Go 1.25.0
- No CI/CD, no Makefile, no linting config
- Uses `github.com/soyacen/gox` (author's utility library)
- 59 Go files, 29 test files across 11 middleware packages
- All 11 packages now have test coverage (recently added)
- Subdirectory AGENTS.md only for `ratelimiter/` and `circuitbreaker/` (highest complexity)

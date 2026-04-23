# grpc-middleware KNOWLEDGE BASE

**Generated:** 2025-01-23
**Commit:** 9ac410f
**Branch:** main

## OVERVIEW

Go gRPC interceptor library providing 11 middleware components: rate limiting (BBR), circuit breaker (SRE), auth, access logging, retry, timeout, error/slow logging, context handling, recovery, and unified error handling. Uses functional options pattern. Comments in Chinese.

## STRUCTURE

```
.
├── ratelimiter/      # BBR algorithm + rolling window + CPU monitoring
├── circuitbreaker/   # SRE algorithm + rolling counter
├── auth/             # Authentication metadata wrapper
├── accesslog/        # Request/response logging (client+server)
├── errorlog/         # Error-only logging
├── slowlog/          # Slow request detection
├── retry/            # Client retry with backoff
├── timeout/          # Unary client timeout (NO stream)
├── context/          # Context manipulation (BUG: stream loses ctx)
├── recovery/         # Panic recovery
├── unifiederror/     # Error response unification
└── internal/         # Empty (implementation exposed at pkg level)
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add new middleware | Create top-level dir/ | Follow `interceptors.go` + `options.go` pattern |
| BBR rate limiting | `ratelimiter/` | CPU-aware, rolling window stats |
| Circuit breaker | `circuitbreaker/` | Google SRE algorithm, client-side only |
| Auth middleware | `auth/` | Metadata-based, wraps handlers |
| Logging middleware | `accesslog/`, `errorlog/`, `slowlog/` | Different log scopes |
| Fix context bug | `context/interceptors.go:91` | Stream interceptor loses modified ctx |

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
| `init` | method | `*/options.go` | Validate/sanitize config |

## CONVENTIONS

- **Package layout**: Each middleware = top-level directory with `interceptors.go`, `options.go`, `*_test.go`
- **Interceptor naming**: `{Stream,Unary}{Client,Server}Interceptor` pattern
- **Options pattern**: `options` struct + `Option` type + `WithXxx()` funcs + `defaultOptions()` + `apply()` + `init()`
- **Comments**: Written in Chinese
- **Root package**: `grpcmiddleware` (declared in `doc.go`, empty otherwise)

## ANTI-PATTERNS (THIS PROJECT)

1. **NEVER** put implementation in `internal/` - it's empty, everything is package-level
2. **NEVER** use inconsistent options init - some use `defaultOptions().apply(opts...).init()`, others break into separate statements
3. **NEVER** shadow stdlib `context` - the `context/` package requires aliased imports
4. **NEVER** assume `timeout` has stream support - only `UnaryClientInterceptor` exists
5. **NEVER** ignore the context stream bug - modified ctx is lost in `context.StreamServerInterceptor`
6. **NEVER** mix `interface{}` and `any` - Go 1.25 project, standardize on `any`
7. **NEVER** panic in constructors - return errors instead (seen in `circuitbreaker/option.go`)

## KNOWN ISSUES

- `context.StreamServerInterceptor`: Wraps stream but doesn't inject modified context (unlike `auth` package)
- `timeout`: Missing `StreamClientInterceptor`
- `internal/container/`: Empty directory, implementation details exposed
- `ratelimiter/cpu.go`: `collectCPU()` goroutine starts via `sync.Once` but never stops
- `accesslog/interceptors.go`: Creates new `sync.Pool` inside every interceptor factory call (defeats pooling purpose)

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

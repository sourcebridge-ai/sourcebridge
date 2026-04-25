---
sourcebridge:
    page_id: test-repo.api_reference
    template: api_reference
    audience: for-engineers
    dependencies:
        paths:
            - '**/*.go'
        symbols:
            - internal/billing.Charge
            - internal/auth.Middleware
        dependency_scope: direct
---
<!-- sourcebridge:block id="b195d13b5c4ff" kind="heading" owner="generated" -->
# API Reference
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b0d2f08a13d03" kind="heading" owner="generated" -->
## internal/auth
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bd48904248a62" kind="paragraph" owner="generated" -->
## Overview
The internal/auth package handles authentication for inbound HTTP requests. (internal/auth/auth.go:1-10)
It validates session tokens and enforces role-based access control on all protected routes.

## Key types
| Type | Purpose |
|---|---|
| Middleware | Wraps http.Handler with session verification (internal/auth/auth.go:12-25) |

## Public API
Middleware is the primary entry point. Pass it any http.Handler to enable authentication. (internal/auth/auth.go:12-25)
RequireRole asserts that the authenticated user holds a given role. (internal/auth/auth.go:30-45)

## Dependencies
- internal/jwt for token parsing (internal/jwt/jwt.go:1-50)
- internal/sessions for session lookup (internal/sessions/sessions.go:1-80)

## Used by
- internal/api/rest (internal/api/rest/rest.go:1-20)
- internal/billing (internal/billing/billing.go:1-20)

## Code example
```go
mux.Handle("/api/", auth.Middleware(apiHandler))
```
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b549c61e4a066" kind="heading" owner="generated" -->
### Middleware
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="baf01a89fb079" kind="code" owner="generated" -->
```go
func Middleware(next http.Handler) http.Handler
```
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bc4d0c3d5995b" kind="paragraph" owner="generated" -->
Middleware wraps an http.Handler with session verification. (internal/auth/auth.go:12-25)
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b0692aedadf16" kind="heading" owner="generated" -->
## internal/billing
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b15a8d3256d32" kind="paragraph" owner="generated" -->
## Overview
The internal/auth package handles authentication for inbound HTTP requests. (internal/auth/auth.go:1-10)
It validates session tokens and enforces role-based access control on all protected routes.

## Key types
| Type | Purpose |
|---|---|
| Middleware | Wraps http.Handler with session verification (internal/auth/auth.go:12-25) |

## Public API
Middleware is the primary entry point. Pass it any http.Handler to enable authentication. (internal/auth/auth.go:12-25)
RequireRole asserts that the authenticated user holds a given role. (internal/auth/auth.go:30-45)

## Dependencies
- internal/jwt for token parsing (internal/jwt/jwt.go:1-50)
- internal/sessions for session lookup (internal/sessions/sessions.go:1-80)

## Used by
- internal/api/rest (internal/api/rest/rest.go:1-20)
- internal/billing (internal/billing/billing.go:1-20)

## Code example
```go
mux.Handle("/api/", auth.Middleware(apiHandler))
```
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bb2660c392c08" kind="heading" owner="generated" -->
### Charge
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b31c2a3cca720" kind="code" owner="generated" -->
```go
func Charge(ctx context.Context, amount int) error
```
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="bb2733e168037" kind="paragraph" owner="generated" -->
Charge initiates a payment. Returns an error when the payment provider rejects the request. (internal/billing/billing.go:5-18)
<!-- /sourcebridge:block -->

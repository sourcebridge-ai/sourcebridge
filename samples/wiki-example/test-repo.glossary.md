---
sourcebridge:
    page_id: test-repo.glossary
    template: glossary
    audience: for-engineers
    dependencies:
        dependency_scope: direct
---
<!-- sourcebridge:block id="bd9045ebcdec1" kind="heading" owner="generated" -->
## internal/auth
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b7517b27f38e6" kind="paragraph" owner="generated" -->
**Middleware** `func Middleware(next http.Handler) http.Handler` (internal/auth/auth.go:12-25)\
Middleware wraps an http.Handler with session verification.
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b38b756a60901" kind="heading" owner="generated" -->
## internal/billing
<!-- /sourcebridge:block -->

<!-- sourcebridge:block id="b7733bf9121b3" kind="paragraph" owner="generated" -->
**Charge** `func Charge(ctx context.Context, amount int) error` (internal/billing/billing.go:5-18)\
Charge initiates a payment. Returns an error when the payment provider rejects the request.
<!-- /sourcebridge:block -->

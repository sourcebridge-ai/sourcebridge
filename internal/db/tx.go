// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package db

import (
	"context"
	"fmt"
	"log/slog"
)

// RunInTx runs fn inside a SurrealDB transaction.
//
// If fn returns an error the transaction is cancelled; if fn panics the
// panic is recovered, the transaction cancelled, and the panic re-raised
// so tests and the runtime still see it.
//
// This is the first use of SurrealDB transactions in SourceBridge. The
// trash feature adopted it to keep cascade soft-deletes atomic, but
// callers should be aware that:
//
//   - SurrealDB's transaction behaviour is not yet battle-tested in this
//     codebase.
//   - There is no savepoint / nested-transaction support; callers must
//     not invoke RunInTx inside another RunInTx.
//   - Large transactions can block the cache; keep scope small.
//
// Callers that need absolute cascade durability should pair this with a
// post-commit reconciler (see internal/trash for the canonical pattern).
func (s *SurrealDB) RunInTx(ctx context.Context, fn func(ctx context.Context) error) (txErr error) {
	db := s.DB()
	if db == nil {
		return fmt.Errorf("database not connected")
	}

	if _, err := s.Query(ctx, "BEGIN TRANSACTION;", nil); err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			if _, err := s.Query(ctx, "CANCEL TRANSACTION;", nil); err != nil {
				slog.Error("cancel transaction after panic failed", "error", err)
			}
			// Re-raise the panic — tests and the runtime need to see it.
			panic(r)
		}
	}()

	if err := fn(ctx); err != nil {
		if _, cancelErr := s.Query(ctx, "CANCEL TRANSACTION;", nil); cancelErr != nil {
			slog.Error("cancel transaction failed", "error", cancelErr, "fn_error", err)
		}
		return err
	}

	if _, err := s.Query(ctx, "COMMIT TRANSACTION;", nil); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

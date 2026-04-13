// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package graphql

import (
	"github.com/sourcebridge/sourcebridge/internal/llm"
	"github.com/sourcebridge/sourcebridge/internal/llm/orchestrator"
)

func appendJobLog(orch *orchestrator.Orchestrator, rt llm.Runtime, level llm.JobLogLevel, phase, event, message string, payload map[string]any) {
	if orch == nil || rt == nil || rt.JobID() == "" {
		return
	}
	_ = orch.AppendJobLog(rt.JobID(), level, phase, event, message, payload)
}

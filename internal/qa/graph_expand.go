// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package qa

// graphAdapterStore is the narrow slice of graph.GraphStore the
// orchestrator calls through. Defined locally so internal/qa doesn't
// import internal/graph directly — the REST server layer wires the
// adapter up at startup.
type graphAdapterStore interface {
	GetCallers(symbolID string) []string
	GetCallees(symbolID string) []string
}

// graphSymbolLookup resolves a symbol ID to its display metadata.
type graphSymbolLookup interface {
	Lookup(id string) (qualifiedName, filePath, language string, startLine, endLine int, ok bool)
}

// NewGraphExpander adapts a GraphStore-shaped collaborator + symbol
// lookup into the orchestrator's GraphExpander interface. The split
// keeps the package boundaries clean: callers at the REST layer pass
// a functional SymbolLookup that reaches into graph.StoredSymbol, and
// this package never imports graph.
func NewGraphExpander(store graphAdapterStore, lookup graphSymbolLookup) GraphExpander {
	return &graphExpander{store: store, lookup: lookup}
}

type graphExpander struct {
	store  graphAdapterStore
	lookup graphSymbolLookup
}

func (g *graphExpander) GetCallers(symbolID string) []GraphNeighbor {
	if g == nil || g.store == nil {
		return nil
	}
	return g.resolve(g.store.GetCallers(symbolID))
}

func (g *graphExpander) GetCallees(symbolID string) []GraphNeighbor {
	if g == nil || g.store == nil {
		return nil
	}
	return g.resolve(g.store.GetCallees(symbolID))
}

func (g *graphExpander) resolve(ids []string) []GraphNeighbor {
	if len(ids) == 0 || g.lookup == nil {
		return nil
	}
	out := make([]GraphNeighbor, 0, len(ids))
	for _, id := range ids {
		qn, fp, lang, start, end, ok := g.lookup.Lookup(id)
		if !ok {
			continue
		}
		out = append(out, GraphNeighbor{
			SymbolID:      id,
			QualifiedName: qn,
			FilePath:      fp,
			StartLine:     start,
			EndLine:       end,
			Language:      lang,
		})
	}
	return out
}

package research

import (
	"slices"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/source"
)

// Pinned documents never change during research. Workers share this index,
// while read ranges and failures remain local to each workspace.
type workspaceSources struct {
	byID  map[string]source.Document
	paths []string
}

func (w *workspace) sourceIndex() *workspaceSources {
	w.sourcesOnce.Do(func() {
		if w.sources != nil {
			return
		}
		index := &workspaceSources{
			byID:  make(map[string]source.Document, len(w.documents)),
			paths: make([]string, 0, len(w.documents)),
		}
		for file, doc := range w.documents {
			index.byID[doc.ID] = doc
			index.paths = append(index.paths, file)
		}
		slices.Sort(index.paths)
		w.sources = index
	})
	return w.sources
}

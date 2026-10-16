package passport

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrInvalidEdit reports a reviewer change outside the editable passport text.
var ErrInvalidEdit = errors.New("invalid passport edit")

// ChapterEdit replaces a chapter title and its Markdown blocks. Diagrams,
// flows, tables and evidence references stay as generated.
type ChapterEdit struct {
	Title    string   `json:"title"`
	Markdown []string `json:"markdown" jsonschema:"Replacement text for each Markdown block, in chapter order."`
}

// EditChapter validates the whole edit before changing the document.
func (d *Document) EditChapter(id string, edit ChapterEdit) error {
	index := slices.IndexFunc(d.Chapters, func(c Chapter) bool { return c.ID == id })
	if index < 0 {
		return fmt.Errorf("%w: unknown chapter", ErrInvalidEdit)
	}
	chapter := &d.Chapters[index]
	if strings.TrimSpace(edit.Title) == "" {
		return fmt.Errorf("%w: the chapter title is required", ErrInvalidEdit)
	}
	var blocks []int
	for i, block := range chapter.Blocks {
		if block.Kind == "markdown" {
			blocks = append(blocks, i)
		}
	}
	if len(blocks) != len(edit.Markdown) {
		return fmt.Errorf("%w: the chapter changed; reload it and try again", ErrInvalidEdit)
	}
	for _, text := range edit.Markdown {
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("%w: text blocks cannot be empty", ErrInvalidEdit)
		}
		if err := validateMarkdown(text); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidEdit, err)
		}
	}
	chapter.Title = strings.TrimSpace(edit.Title)
	for i, block := range blocks {
		chapter.Blocks[block].Markdown = edit.Markdown[i]
	}
	return nil
}

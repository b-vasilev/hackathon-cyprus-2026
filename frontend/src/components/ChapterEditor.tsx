import { useMutation, useQueryClient } from '@tanstack/react-query';
import { RefreshCw } from 'lucide-react';
import { useState } from 'react';
import { api, type Chapter, type DocumentBlock, type Revision } from '../lib/api';
import { ErrorNotice } from './ui';

const blockLabels: Record<string, string> = {
  diagram_ref: 'Diagram',
  scenario_ref: 'Flow',
  object_table: 'Table',
};

/** Edits a chapter's title and text. Generated diagrams, flows and tables stay as they are. */
export function ChapterEditor({
  revision,
  chapter,
  onDone,
}: Readonly<{ revision: Revision; chapter: Chapter; onDone: () => void }>) {
  const queryClient = useQueryClient();
  const blocks: DocumentBlock[] = chapter.blocks ?? [];
  const [title, setTitle] = useState(chapter.title);
  const [texts, setTexts] = useState(() =>
    blocks.filter((block) => block.kind === 'markdown').map((block) => block.markdown ?? '')
  );
  const save = useMutation({
    mutationFn: () =>
      api<Revision>(`/passport-revisions/${revision.id}/chapters/${chapter.id}`, {
        method: 'PUT',
        body: JSON.stringify({ title, markdown: texts }),
      }),
    onSuccess: async (saved) => {
      queryClient.setQueryData(['revision', saved.id], saved);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['review', saved.id] }),
        queryClient.invalidateQueries({ queryKey: ['revisions', saved.product_id] }),
      ]);
      onDone();
    },
  });
  // Each Markdown block maps to its position in the edited text list.
  const textPositions = blocks.map(
    (_, index) => blocks.slice(0, index).filter((block) => block.kind === 'markdown').length
  );
  return (
    <form
      className="passport-section chapter-editor"
      aria-label={`Edit ${chapter.title}`}
      onSubmit={(event) => {
        event.preventDefault();
        save.mutate();
      }}
    >
      <label>
        Chapter title
        <input value={title} required onChange={(event) => setTitle(event.target.value)} />
      </label>
      {blocks.map((block, index) => {
        if (block.kind !== 'markdown') {
          return (
            // eslint-disable-next-line react/no-array-index-key -- Blocks are identified by their authored position.
            <p className="chapter-editor-fixed" key={`${chapter.id}:${index}`}>
              {blockLabels[block.kind] ?? 'Generated content'} · generated, not editable here
            </p>
          );
        }
        const position = textPositions[index] ?? 0;
        return (
          // eslint-disable-next-line react/no-array-index-key -- Blocks are identified by their authored position.
          <label key={`${chapter.id}:${index}`}>
            Text {position + 1}
            <textarea
              value={texts[position]}
              required
              rows={Math.min(24, Math.max(4, (texts[position] ?? '').split('\n').length + 1))}
              onChange={(event) =>
                setTexts((current) =>
                  current.map((text, i) => (i === position ? event.target.value : text))
                )
              }
            />
          </label>
        );
      })}
      <p className="meta">
        Markdown is supported. Raw HTML is not allowed.
        {revision.published && ' Learners see your changes after you publish them.'}
      </p>
      {save.isError && <ErrorNotice error={save.error} />}
      <div className="review-note-actions">
        <button type="button" className="button quiet" disabled={save.isPending} onClick={onDone}>
          Cancel
        </button>
        <button type="submit" className="button primary" disabled={save.isPending}>
          {save.isPending && <RefreshCw size={16} className="spin" aria-hidden="true" />}
          {save.isPending ? 'Saving…' : 'Save chapter'}
        </button>
      </div>
    </form>
  );
}

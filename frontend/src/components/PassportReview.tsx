import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, EyeOff, Flag, MessageSquare, RefreshCw } from 'lucide-react';
import { useState } from 'react';
import {
  api,
  date,
  type LearningPack,
  type Review,
  type ReviewActivity,
  type Revision,
} from '../lib/api';
import { packKey } from '../lib/learningPack';
import { DisclosureHint, ErrorNotice } from './ui';

interface ChapterOption {
  id: string;
  title: string;
}

/** Lead and admin controls for one passport version. Learners never see this panel. */
export function PassportReview({
  revision,
  chapters,
  currentChapterId,
}: Readonly<{ revision: Revision; chapters: ChapterOption[]; currentChapterId: string }>) {
  const queryClient = useQueryClient();
  const review = useQuery({
    queryKey: ['review', revision.id],
    queryFn: ({ signal }) => api<Review>(`/passport-revisions/${revision.id}/review`, { signal }),
  });
  const [flagging, setFlagging] = useState(false);
  const practice = useQuery({
    queryKey: [packKey, revision.id],
    queryFn: ({ signal }) =>
      api<LearningPack>(`/passport-revisions/${revision.id}/learning-pack`, { signal }),
    refetchInterval: (query) => (query.state.data?.status === 'generating' ? 4000 : false),
  });
  const refresh = async (productId: string) => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['review', revision.id] }),
      queryClient.invalidateQueries({ queryKey: ['revisions', productId] }),
      queryClient.invalidateQueries({ queryKey: ['published-revision'] }),
      queryClient.invalidateQueries({ queryKey: ['product', productId] }),
      queryClient.invalidateQueries({ queryKey: ['products'] }),
      queryClient.invalidateQueries({ queryKey: [packKey] }),
    ]);
  };
  const publication = useMutation({
    mutationFn: (action: 'publish' | 'unpublish') =>
      api<Revision>(`/passport-revisions/${revision.id}/${action}`, { method: 'POST' }),
    onSuccess: async (saved) => {
      queryClient.setQueryData(['revision', saved.id], saved);
      await refresh(saved.product_id);
    },
  });
  const flag = useMutation({
    mutationFn: (reason: string | null) =>
      api<Review>(`/passport-revisions/${revision.id}/flag`, {
        method: reason === null ? 'DELETE' : 'PUT',
        body: reason === null ? undefined : JSON.stringify({ reason }),
      }),
    onSuccess: async (saved) => {
      queryClient.setQueryData(['review', revision.id], saved);
      setFlagging(false);
      await refresh(revision.product_id);
    },
  });
  const flagged = (review.data?.flagged_at ?? '') !== '';
  const busy = publication.isPending || flag.isPending;
  const readyPractice =
    practice.data?.status === 'ready' && !practice.data.stale && Boolean(practice.data.content);
  const canPublish =
    !revision.published ||
    revision.unpublished_changes ||
    (readyPractice && (!practice.data?.published || practice.data.unpublished_changes));
  let title = `Draft · version ${revision.version}`;
  let description = 'Learners cannot see this version. Review, edit and publish it when ready.';
  if (revision.published) {
    title = `Published · version ${revision.version}`;
    description = revision.unpublished_changes
      ? 'Learners still see the earlier published text. Publish the changes to update it.'
      : `Learners see this version. Published ${date(revision.published_at)}.`;
  }
  return (
    <section className="passport-review" aria-label="Passport review">
      <div className="passport-publication">
        <div aria-live="polite">
          <h2>{title}</h2>
          <p>{description}</p>
          {readyPractice && <p>Publishing this passport also publishes its quiz and first task.</p>}
          {practice.data?.stale && (
            <p>
              The quiz needs updating after the text changes. Publish now without it, or generate
              practice again first.
            </p>
          )}
          {practice.data?.status === 'generating' && (
            <p>The quiz is still generating. You can publish both when it is ready.</p>
          )}
          {review.data !== undefined && review.data.edited_at !== '' && (
            <p>
              Last edited {date(review.data.edited_at)}
              {review.data.edited_by === '' ? '' : ` by ${review.data.edited_by}`}.
            </p>
          )}
        </div>
        <div className="review-actions">
          {flagged ? (
            <button
              type="button"
              className="button quiet"
              disabled={busy}
              onClick={() => flag.mutate(null)}
            >
              <Check size={16} aria-hidden="true" />
              Resolve flag
            </button>
          ) : (
            <button
              type="button"
              className="button quiet"
              disabled={busy || flagging}
              onClick={() => setFlagging(true)}
            >
              <Flag size={16} aria-hidden="true" />
              Flag for review
            </button>
          )}
          {revision.published && (
            <button
              type="button"
              className="button quiet"
              disabled={busy}
              onClick={() => publication.mutate('unpublish')}
            >
              <EyeOff size={16} aria-hidden="true" />
              Unpublish
            </button>
          )}
          {canPublish && (
            <button
              type="button"
              className="button primary"
              disabled={
                busy || flagged || review.isPending || practice.data?.status === 'generating'
              }
              onClick={() => publication.mutate('publish')}
            >
              {publication.isPending ? (
                <RefreshCw size={16} className="spin" aria-hidden="true" />
              ) : (
                <Check size={16} aria-hidden="true" />
              )}
              {revision.published ? 'Publish changes' : `Publish version ${revision.version}`}
            </button>
          )}
        </div>
      </div>
      {flagging && (
        <NoteForm
          label="What needs review?"
          submitLabel="Flag for review"
          pending={flag.isPending}
          onCancel={() => setFlagging(false)}
          onSubmit={(reason) => flag.mutate(reason)}
        />
      )}
      {flagged && review.data !== undefined && (
        <div className="notice warning" role="status">
          <Flag size={18} aria-hidden="true" />
          <div>
            <strong>
              Flagged for review
              {review.data.flagged_by === '' ? '' : ` by ${review.data.flagged_by}`} ·{' '}
              {date(review.data.flagged_at)}
            </strong>
            <p>{review.data.flag_reason}</p>
            <p>Resolve the flag before publishing this version.</p>
          </div>
        </div>
      )}
      {publication.isError && <ErrorNotice error={publication.error} />}
      {flag.isError && <ErrorNotice error={flag.error} />}
      {review.isError && <ErrorNotice error={review.error} />}
      <ReviewComments
        revisionId={revision.id}
        activity={review.data?.activity ?? []}
        chapters={chapters}
        currentChapterId={currentChapterId}
      />
    </section>
  );
}

function ReviewComments({
  revisionId,
  activity,
  chapters,
  currentChapterId,
}: Readonly<{
  revisionId: string;
  activity: ReviewActivity[];
  chapters: ChapterOption[];
  currentChapterId: string;
}>) {
  const queryClient = useQueryClient();
  const [chapterId, setChapterId] = useState<string | null>(null);
  const [formKey, setFormKey] = useState(0);
  const comment = useMutation({
    mutationFn: (body: string) =>
      api<Review>(`/passport-revisions/${revisionId}/comments`, {
        method: 'POST',
        body: JSON.stringify({ chapter_id: chapterId ?? currentChapterId, body }),
      }),
    onSuccess: (saved) => {
      queryClient.setQueryData(['review', revisionId], saved);
      setFormKey((key) => key + 1);
    },
  });
  const titles = new Map(chapters.map((item) => [item.id, item.title]));
  const comments = activity.filter((item) => item.kind === 'comment').length;
  return (
    <details className="sources-panel review-comments" open={comments > 0 || undefined}>
      <summary>
        <MessageSquare size={17} aria-hidden="true" />
        Review comments ({comments})
        <DisclosureHint show="Show comments" hide="Hide comments" />
      </summary>
      {activity.length === 0 ? (
        <p className="meta">No comments or review activity on this version yet.</p>
      ) : (
        <ol className="review-activity">
          {activity.map((item) => (
            <li key={item.id} className={item.kind === 'comment' ? 'comment' : ''}>
              <div className="review-activity-meta">
                <strong>{item.author === '' ? 'Deleted account' : item.author}</strong>
                <time dateTime={item.created_at}>{date(item.created_at)}</time>
                {item.chapter_id !== '' && (
                  <span className="badge">{titles.get(item.chapter_id) ?? item.chapter_id}</span>
                )}
              </div>
              <p>{activityText(item)}</p>
            </li>
          ))}
        </ol>
      )}
      <label className="review-comment-chapter">
        About
        <select
          value={chapterId ?? currentChapterId}
          onChange={(event) => setChapterId(event.target.value)}
        >
          <option value="">Whole passport</option>
          {chapters.map((item) => (
            <option key={item.id} value={item.id}>
              {item.title}
            </option>
          ))}
        </select>
      </label>
      <NoteForm
        key={formKey}
        label="Comment"
        submitLabel="Add comment"
        pending={comment.isPending}
        onSubmit={(body) => comment.mutate(body)}
      />
      {comment.isError && <ErrorNotice error={comment.error} />}
    </details>
  );
}

function activityText(item: ReviewActivity): string {
  switch (item.kind) {
    case 'comment':
      return item.body;
    case 'edit':
      return 'Edited this chapter.';
    case 'flag':
      return `Flagged for review: ${item.body}`;
    case 'resolve':
      return 'Resolved the review flag.';
    case 'publish':
      return 'Published this version.';
    case 'unpublish':
      return 'Unpublished this version.';
    default:
      return item.body;
  }
}

function NoteForm({
  label,
  submitLabel,
  pending,
  onSubmit,
  onCancel,
}: Readonly<{
  label: string;
  submitLabel: string;
  pending: boolean;
  onSubmit: (text: string) => void;
  onCancel?: () => void;
}>) {
  const [text, setText] = useState('');
  return (
    <form
      className="review-note-form"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit(text);
      }}
    >
      <label>
        {label}
        <textarea
          value={text}
          rows={3}
          maxLength={4000}
          required
          onChange={(event) => setText(event.target.value)}
        />
      </label>
      <div className="review-note-actions">
        {onCancel !== undefined && (
          <button type="button" className="button quiet" disabled={pending} onClick={onCancel}>
            Cancel
          </button>
        )}
        <button type="submit" className="button primary" disabled={pending || text.trim() === ''}>
          {pending ? 'Saving…' : submitLabel}
        </button>
      </div>
    </form>
  );
}

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Check, RefreshCw } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { api, date, type Revision, type RevisionInfo } from '../lib/api';
import { ErrorNotice } from './ui';

export function PassportPublication({
  revision,
  history,
}: Readonly<{ revision: Revision; history: RevisionInfo[] }>) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const update = useMutation({
    mutationFn: (action: 'approve' | 'publish') =>
      api<Revision>(`/passport-revisions/${revision.id}/${action}`, { method: 'POST' }),
    onSuccess: async (saved, action) => {
      queryClient.setQueryData(['revision', saved.id], saved);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['revision'] }),
        queryClient.invalidateQueries({ queryKey: ['published-revision'] }),
        queryClient.invalidateQueries({ queryKey: ['revisions', saved.product_id] }),
        queryClient.invalidateQueries({ queryKey: ['product', saved.product_id] }),
        queryClient.invalidateQueries({ queryKey: ['products'] }),
      ]);
      if (action === 'publish') {
        void navigate(`/published/${saved.product_id}`);
      }
    },
  });
  const approved = revision.approved_at !== '';
  const published = history.find((item) => item.published);
  let actionLabel = `Approve version ${revision.version}`;
  if (approved) {
    actionLabel = `Publish version ${revision.version}`;
  }
  if (revision.published) {
    actionLabel = `Version ${revision.version} is published`;
  }
  let title = 'Review this draft';
  if (revision.published) {
    title = 'Published passport';
  } else if (approved) {
    title = 'Ready to publish';
  }
  return (
    <>
      <section className="passport-publication" aria-label="Passport approval and publishing">
        <div aria-live="polite">
          <h2>{title}</h2>
          {revision.published ? (
            <p>
              Version {revision.version} was published {date(revision.published_at)}.
            </p>
          ) : (
            <p>
              {approved
                ? `Version ${revision.version} was approved ${date(revision.approved_at)}.`
                : 'Review the passport and source warnings before approving this version.'}
            </p>
          )}
          {!revision.published && (
            <p>
              {published === undefined
                ? 'No version is published yet.'
                : `Version ${published.version} stays published until you publish a replacement.`}
            </p>
          )}
        </div>
        <button
          type="button"
          className="button primary"
          disabled={update.isPending || revision.published}
          onClick={() => update.mutate(approved ? 'publish' : 'approve')}
        >
          {update.isPending ? (
            <RefreshCw size={16} className="spin" aria-hidden="true" />
          ) : (
            <Check size={16} aria-hidden="true" />
          )}
          {update.isPending ? 'Saving…' : actionLabel}
        </button>
      </section>
      {update.isError && <ErrorNotice error={update.error} />}
    </>
  );
}

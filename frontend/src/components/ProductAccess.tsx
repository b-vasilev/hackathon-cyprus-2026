import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Trash2, Users } from 'lucide-react';
import { useId, useState, type FormEvent } from 'react';
import { api, type Member, type ProductSharing, type User } from '../lib/api';
import { isAdmin, useMe } from '../lib/auth';
import { DisclosureHint, ErrorNotice, Loading } from './ui';

/** Lets admins and product leads pick who can lead or learn a product. */
export function ProductAccess({
  productId,
  title = 'People and access',
}: Readonly<{ productId: string; title?: string }>) {
  const me = useMe();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('learner');
  const sharingHelpId = useId();
  // Load members only once the panel is opened, so long product lists stay cheap.
  const members = useQuery({
    queryKey: ['members', productId],
    queryFn: ({ signal }) => api<Member[]>(`/products/${productId}/members`, { signal }),
    enabled: open,
  });
  const users = useQuery({
    queryKey: ['users'],
    queryFn: ({ signal }) => api<User[]>('/users', { signal }),
    enabled: open,
  });
  const sharing = useQuery({
    queryKey: ['product-sharing', productId],
    queryFn: ({ signal }) => api<ProductSharing>(`/products/${productId}/sharing`, { signal }),
    enabled: open,
  });
  const candidates =
    users.data?.filter(
      (user) => members.data?.some((member) => member.user_id === user.id) !== true
    ) ?? [];
  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: ['members', productId] }),
      queryClient.invalidateQueries({ queryKey: ['me'] }),
    ]);
  const save = useMutation({
    mutationFn: (input: Readonly<{ email: string; role: string }>) =>
      api<Member>(`/products/${productId}/members`, {
        method: 'PUT',
        body: JSON.stringify(input),
      }),
    onSuccess: async (_, input) => {
      if (input.email === email) {
        setEmail('');
      }
      await refresh();
    },
  });
  const remove = useMutation({
    mutationFn: (userId: string) =>
      api<undefined>(`/products/${productId}/members/${userId}`, { method: 'DELETE' }),
    onSuccess: refresh,
  });
  const share = useMutation({
    mutationFn: (sharedWithAll: boolean) =>
      api<ProductSharing>(`/products/${productId}/sharing`, {
        method: 'PUT',
        body: JSON.stringify({ shared_with_all: sharedWithAll }),
      }),
    onSuccess: async (value) => {
      queryClient.setQueryData(['product-sharing', productId], value);
      await refresh();
    },
  });
  function submit(event: FormEvent) {
    event.preventDefault();
    save.mutate({ email, role });
  }
  const busy = save.isPending || remove.isPending || share.isPending;
  return (
    <details className="sources-panel" onToggle={(event) => setOpen(event.currentTarget.open)}>
      <summary>
        <Users size={17} aria-hidden="true" />
        {title}
        <DisclosureHint show="Manage access" hide="Hide access" />
      </summary>
      <p className="field-help access-help">
        Product leads generate the passport and manage access. Learners read the product and its
        published passport. People must register before you can add them individually.
      </p>
      {sharing.isLoading && <Loading label="Loading sharing settings…" />}
      {sharing.isError && <ErrorNotice error={sharing.error} />}
      {sharing.data && (
        <div className="sharing-policy">
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={sharing.data.shared_with_all}
              disabled={!isAdmin(me) || busy}
              aria-describedby={sharingHelpId}
              onChange={(event) => share.mutate(event.target.checked)}
            />
            All registered users
          </label>
          <p className="field-help" id={sharingHelpId}>
            Share this product and its published passport with every registered user as a learner.
            New accounts get access automatically. Turning this off keeps individual access.
            {!isAdmin(me) && ' Only admins can change this setting.'}
          </p>
          {sharing.data.shared_with_all && (
            <p className="meta" role="status">
              All registered users have learner access, including future accounts.
            </p>
          )}
        </div>
      )}
      {share.isError && <ErrorNotice error={share.error} />}
      {members.isLoading && <Loading label="Loading people…" />}
      {members.isError && <ErrorNotice error={members.error} />}
      {users.isError && <ErrorNotice error={users.error} />}
      {members.data?.length === 0 && sharing.data?.shared_with_all === false && (
        <p className="meta">Only admins can open this product.</p>
      )}
      <ul className="people-list" aria-label="Product members">
        {members.data?.map((member) => (
          <li className="person-row" key={member.user_id}>
            <strong>
              {member.email}
              {member.user_id === me.user.id && ' (you)'}
            </strong>
            <label className="inline-label">
              Role
              <select
                value={member.role}
                disabled={busy}
                onChange={(event) => save.mutate({ email: member.email, role: event.target.value })}
              >
                <option value="lead">Product lead</option>
                <option value="learner">Learner</option>
              </select>
            </label>
            <button
              type="button"
              className="text-button destructive"
              disabled={busy}
              onClick={() => remove.mutate(member.user_id)}
            >
              <Trash2 size={15} aria-hidden="true" />
              Remove
            </button>
          </li>
        ))}
      </ul>
      <form className="member-form" onSubmit={submit}>
        <label>
          Person
          <select
            required
            value={email}
            disabled={users.isPending}
            onChange={(event) => setEmail(event.target.value)}
          >
            <option value="" disabled>
              {candidates.length === 0 && users.isSuccess
                ? 'Everyone already has access'
                : 'Choose a person'}
            </option>
            {candidates.map((user) => (
              <option key={user.id} value={user.email}>
                {user.email}
                {user.role === 'admin' ? ' (admin)' : ''}
              </option>
            ))}
          </select>
        </label>
        <label>
          Role
          <select value={role} onChange={(event) => setRole(event.target.value)}>
            <option value="learner">Learner</option>
            <option value="lead">Product lead</option>
          </select>
        </label>
        <button className="button secondary" disabled={busy}>
          Grant access
        </button>
      </form>
      {save.isError && <ErrorNotice error={save.error} />}
      {remove.isError && <ErrorNotice error={remove.error} />}
    </details>
  );
}

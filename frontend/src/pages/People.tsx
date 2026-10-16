import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ShieldCheck, Trash2 } from 'lucide-react';
import { ProductAccess } from '../components/ProductAccess';
import { ErrorNotice, Loading, PageHeading } from '../components/ui';
import { api, date, type Product, type User } from '../lib/api';
import { useMe } from '../lib/auth';

export function People() {
  const me = useMe();
  const queryClient = useQueryClient();
  const users = useQuery({
    queryKey: ['users'],
    queryFn: ({ signal }) => api<User[]>('/users', { signal }),
  });
  const products = useQuery({
    queryKey: ['products'],
    queryFn: ({ signal }) => api<Product[]>('/products', { signal }),
  });
  // Refresh the signed-in user too, in case an admin changed their own role.
  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: ['users'] }),
      queryClient.invalidateQueries({ queryKey: ['me'] }),
      // Removing an account also removes its product roles.
      queryClient.invalidateQueries({ queryKey: ['members'] }),
    ]);
  const setRole = useMutation({
    mutationFn: ({ id, role }: Readonly<{ id: string; role: string }>) =>
      api<User>(`/users/${id}`, { method: 'PATCH', body: JSON.stringify({ role }) }),
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: (id: string) => api<undefined>(`/users/${id}`, { method: 'DELETE' }),
    onSuccess: refresh,
  });
  return (
    <>
      <PageHeading
        eyebrow="Wayfinder settings"
        title="People and roles"
        description="Admins have full access. Everyone else sees only the products they were given."
      />
      <div className="intro-strip">
        <ShieldCheck size={19} aria-hidden="true" />
        <p>
          Product leads manage a product, its passport and its people. Learners read the product and
          its passport. One person can hold different roles on different products.
        </p>
      </div>
      <div className="people-layout">
        <section className="people-section" aria-labelledby="accounts-heading">
          <h2 className="people-section-heading" id="accounts-heading">
            Accounts
          </h2>
          {users.isPending && <Loading label="Loading people…" />}
          {users.isError && <ErrorNotice error={users.error} />}
          {setRole.isError && <ErrorNotice error={setRole.error} />}
          {remove.isError && <ErrorNotice error={remove.error} />}
          <ul className="people-list" aria-labelledby="accounts-heading">
            {users.data?.map((user) => {
              const self = user.id === me.user.id;
              return (
                <li className="person-row" key={user.id}>
                  <div>
                    <strong>
                      {user.email}
                      {self && ' (you)'}
                    </strong>
                    <span className="meta">Joined {date(user.created_at)}</span>
                  </div>
                  <label className="checkbox-label inline-label">
                    <input
                      type="checkbox"
                      checked={user.role === 'admin'}
                      disabled={setRole.isPending}
                      aria-label={`Admin: ${user.email}`}
                      onChange={(event) =>
                        setRole.mutate({
                          id: user.id,
                          role: event.target.checked ? 'admin' : 'user',
                        })
                      }
                    />
                    Admin
                  </label>
                  <button
                    type="button"
                    className="text-button destructive"
                    disabled={remove.isPending}
                    onClick={() => {
                      if (globalThis.confirm(`Remove ${user.email} and all of their access?`)) {
                        remove.mutate(user.id);
                      }
                    }}
                  >
                    <Trash2 size={15} aria-hidden="true" />
                    Remove
                  </button>
                </li>
              );
            })}
          </ul>
        </section>
        <section className="people-section" aria-labelledby="product-access-heading">
          <h2 className="people-section-heading" id="product-access-heading">
            Product access
          </h2>
          <p className="page-description">
            Choose the product leads and learners for each product. Admins already have access to
            every product.
          </p>
          {products.isPending && <Loading label="Loading products…" />}
          {products.isError && <ErrorNotice error={products.error} />}
          {products.data?.length === 0 && <p className="meta">No products yet.</p>}
          <div className="product-access-list">
            {products.data?.map((product) => (
              <ProductAccess key={product.id} productId={product.id} title={product.name} />
            ))}
          </div>
        </section>
      </div>
    </>
  );
}

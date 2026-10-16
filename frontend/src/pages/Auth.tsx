import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowRight } from 'lucide-react';
import { useState, type FormEvent } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { ErrorNotice, PageHeading } from '../components/ui';
import { api, type Me } from '../lib/api';
import { setSignedInUser } from '../lib/auth';

export function AuthPage({ mode }: Readonly<{ mode: 'login' | 'register' }>) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const location = useLocation();
  // A signed-out visitor to a deep link (an AI app's consent request, a shared passport)
  // returns there after signing in, with its query string intact.
  // Switching between sign-in and registration carries the destination along.
  const carried = (location.state as { from?: string } | null)?.from;
  const destination =
    carried ??
    (['/login', '/register'].includes(location.pathname)
      ? '/'
      : `${location.pathname}${location.search}${location.hash}`);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const registering = mode === 'register';
  const submit = useMutation({
    mutationFn: () =>
      api<Me>(registering ? '/auth/register' : '/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password }),
      }),
    // The app shell renders the signed-in routes once the user is known.
    onSuccess: async (me) => {
      await setSignedInUser(queryClient, me);
      void navigate(destination, {
        replace: true,
        state: {
          authMessage: registering
            ? 'Account created. You are signed in.'
            : 'Signed in successfully.',
        },
      });
    },
  });
  function onSubmit(event: FormEvent) {
    event.preventDefault();
    submit.mutate();
  }
  return (
    <div className="auth-page">
      <PageHeading
        eyebrow={registering ? 'New account' : 'Welcome back'}
        title={registering ? 'Create your account.' : 'Sign in to Wayfinder.'}
        description={
          registering
            ? 'An admin or product lead gives you access to products.'
            : 'Open your Wayfinder workspace.'
        }
      />
      <form className="form-panel" onSubmit={onSubmit}>
        <label>
          Email
          <input
            type="email"
            required
            autoComplete="email"
            maxLength={254}
            value={email}
            onChange={(event) => setEmail(event.target.value)}
          />
        </label>
        <label>
          Password
          <input
            type="password"
            required
            autoComplete={registering ? 'new-password' : 'current-password'}
            minLength={registering ? 8 : undefined}
            maxLength={72}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            aria-describedby={registering ? 'password-help' : undefined}
          />
        </label>
        {registering && (
          <p id="password-help" className="field-help">
            Use 8–72 characters.
          </p>
        )}
        {submit.isError && <ErrorNotice error={submit.error} />}
        <div className="form-actions">
          <Link
            className="text-link"
            to={registering ? '/login' : '/register'}
            state={{ from: destination }}
          >
            {registering ? 'I already have an account' : 'Create an account'}
          </Link>
          <button className="button primary" disabled={submit.isPending}>
            {registering ? 'Create account' : 'Sign in'}
            <ArrowRight size={17} />
          </button>
        </div>
      </form>
    </div>
  );
}

import { useQuery } from '@tanstack/react-query';
import { LogOut } from 'lucide-react';
import { Navigate, NavLink, Route, Routes } from 'react-router-dom';
import { AchievementCenter } from './components/Achievements';
import { Badge, Brand, ErrorNotice, Loading } from './components/ui';
import { api, ApiError, type Me } from './lib/api';
import { isAdmin, MeContext } from './lib/auth';
import { AuthPage } from './pages/Auth';
import { Connections } from './pages/Connections';
import { FoundationAchievements, FoundationModulePage, Foundations } from './pages/Foundations';
import { NotFound } from './pages/NotFound';
import { PassportPage } from './pages/Passport';
import { People } from './pages/People';
import { NewProduct, Products } from './pages/Products';

function signOut() {
  void api<undefined>('/auth/logout', { method: 'POST' }).finally(() =>
    globalThis.location.assign('/login')
  );
}

function SignedInApp({ me }: Readonly<{ me: Me }>) {
  const admin = isAdmin(me);
  return (
    <MeContext value={me}>
      <header className="site-header">
        <div className="header-inner">
          <Brand />
          <nav aria-label="Main navigation">
            <NavLink to="/" end>
              Product spaces
            </NavLink>
            <NavLink to="/foundations">Foundations</NavLink>
            {admin && <NavLink to="/connections">Connections</NavLink>}
            {admin && <NavLink to="/access">People</NavLink>}
          </nav>
          <div className="header-account">
            <span className="account-email">
              {me.user.email}
              {admin && <Badge>Admin</Badge>}
            </span>
            <button type="button" className="button quiet" onClick={signOut}>
              <LogOut size={16} aria-hidden="true" />
              Sign out
            </button>
          </div>
        </div>
      </header>
      <main className="main-content">
        <Routes>
          <Route path="/" element={<Products />} />
          {admin && <Route path="/products/new" element={<NewProduct />} />}
          <Route path="/products/:id" element={<PassportPage />} />
          <Route path="/foundations" element={<Foundations />} />
          <Route path="/foundations/achievements" element={<FoundationAchievements />} />
          <Route path="/foundations/:id" element={<FoundationModulePage />} />
          {admin && <Route path="/connections" element={<Connections />} />}
          {admin && <Route path="/access" element={<People />} />}
          <Route path="/login" element={<Navigate to="/" replace />} />
          <Route path="/register" element={<Navigate to="/" replace />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>
      <AchievementCenter />
    </MeContext>
  );
}

export function App() {
  const me = useQuery({
    queryKey: ['me'],
    queryFn: ({ signal }) => api<Me>('/auth/me', { signal }),
  });
  if (me.isPending) {
    return <Loading label="Opening Wayfinder…" />;
  }
  if (me.isSuccess) {
    return <SignedInApp me={me.data} />;
  }
  if (!(me.error instanceof ApiError && me.error.status === 401)) {
    return <ErrorNotice error={me.error} />;
  }
  return (
    <>
      <header className="site-header">
        <div className="header-inner">
          <Brand />
          <span className="header-caption">Know the product. Find your way.</span>
        </div>
      </header>
      <main className="main-content">
        <Routes>
          <Route path="/register" element={<AuthPage mode="register" />} />
          <Route path="*" element={<AuthPage mode="login" />} />
        </Routes>
      </main>
    </>
  );
}

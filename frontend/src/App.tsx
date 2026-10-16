import { useQuery } from '@tanstack/react-query';
import { LogOut, Zap } from 'lucide-react';
import { lazy, Suspense, useEffect, useState } from 'react';
import {
  Link,
  Navigate,
  NavLink,
  Route,
  Routes,
  useLocation,
  useNavigate,
  useParams,
} from 'react-router-dom';
import { AchievementCenter } from './components/Achievements';
import { Badge, Brand, ErrorNotice, Loading } from './components/ui';
import { flushLearning, syncLearning } from './foundations/sync';
import { api, ApiError, type Me } from './lib/api';
import { canManage, canUseStudio, isAdmin, MeContext, useMe } from './lib/auth';
import { NotFound } from './pages/NotFound';

const AuthPage = lazy(() => import('./pages/Auth').then((page) => ({ default: page.AuthPage })));
const AIConnections = lazy(() =>
  import('./pages/AiConnections').then((page) => ({ default: page.AIConnections }))
);
const AIConsent = lazy(() =>
  import('./pages/AiConnections').then((page) => ({ default: page.AIConsent }))
);
const Connections = lazy(() =>
  import('./pages/Connections').then((page) => ({ default: page.Connections }))
);
const Foundations = lazy(() =>
  import('./pages/Foundations').then((page) => ({ default: page.Foundations }))
);
const FoundationAchievements = lazy(() =>
  import('./pages/Foundations').then((page) => ({ default: page.FoundationAchievements }))
);
const FoundationModulePage = lazy(() =>
  import('./pages/Foundations').then((page) => ({ default: page.FoundationModulePage }))
);
const Home = lazy(() => import('./pages/Home').then((page) => ({ default: page.Home })));
const Landing = lazy(() => import('./pages/Landing').then((page) => ({ default: page.Landing })));
const PassportPage = lazy(() =>
  import('./pages/Passport').then((page) => ({ default: page.PassportPage }))
);
const People = lazy(() => import('./pages/People').then((page) => ({ default: page.People })));
const NewProduct = lazy(() =>
  import('./pages/Products').then((page) => ({ default: page.NewProduct }))
);
const Products = lazy(() =>
  import('./pages/Products').then((page) => ({ default: page.Products }))
);
const PublishedPassports = lazy(() =>
  import('./pages/Published').then((page) => ({ default: page.PublishedPassports }))
);

function signOut() {
  // Save a pending progress change while the session is still valid.
  void flushLearning()
    .then(() => api<undefined>('/auth/logout', { method: 'POST' }))
    .finally(() => globalThis.location.assign('/'));
}

function SignedInApp({ me }: Readonly<{ me: Me }>) {
  const location = useLocation();
  const navigate = useNavigate();
  // Shown once after signing in; clearing the history state keeps it from returning on reload.
  const [authMessage, setAuthMessage] = useState(
    () => (location.state as { authMessage?: string } | null)?.authMessage
  );
  useEffect(() => {
    if (authMessage !== undefined) {
      void navigate(`${location.pathname}${location.search}`, { replace: true, state: null });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- runs once on mount; the message is already captured in state
  }, []);
  const admin = isAdmin(me);
  const studio = canUseStudio(me);
  // Learning progress lives in the account; pages render once the browser copy is merged with it.
  const learning = useQuery({
    queryKey: ['learning', me.user.id],
    queryFn: () => syncLearning(me.user.id),
    staleTime: Number.POSITIVE_INFINITY,
  });
  if (learning.isPending) {
    return <Loading label="Loading your progress…" />;
  }
  return (
    <MeContext value={me}>
      <header className="site-header">
        <div className="header-inner">
          <Brand />
          <nav aria-label="Main navigation">
            <NavLink to="/published">Products</NavLink>
            {studio && <NavLink to="/products">Studio</NavLink>}
            <NavLink to="/foundations">Foundations</NavLink>
            <NavLink to="/connections">Connections</NavLink>
            {admin && <NavLink to="/access">People</NavLink>}
          </nav>
          <div className="header-account">
            <NavLink
              className="button connect-ai-link"
              to="/ai-connections"
              aria-label="MCP"
              title="Connect AI with MCP"
            >
              <Zap size={14} aria-hidden="true" />
              <span>MCP</span>
            </NavLink>
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
        {authMessage !== undefined && authMessage !== '' && (
          <div className="notice" role="status">
            {authMessage}
            <button type="button" className="text-button" onClick={() => setAuthMessage(undefined)}>
              Dismiss
            </button>
          </div>
        )}
        <Suspense fallback={<Loading label="Loading page…" />}>
          <Routes>
            <Route path="/ai-connections" element={<AIConnections />} />
            <Route path="/ai-connections/authorize" element={<AIConsent />} />
            <Route path="/" element={<Home />} />
            <Route path="/published" element={<PublishedPassports />} />
            <Route path="/published/:id" element={<PassportPage key="published" publishedView />} />
            {studio && <Route path="/products" element={<Products />} />}
            {admin && <Route path="/products/new" element={<NewProduct />} />}
            {studio && <Route path="/products/:id" element={<StudioPassport />} />}
            <Route path="/foundations" element={<Foundations />} />
            <Route path="/foundations/achievements" element={<FoundationAchievements />} />
            <Route path="/foundations/:id" element={<FoundationModulePage />} />
            <Route path="/connections" element={<Connections />} />
            {admin && <Route path="/access" element={<People />} />}
            <Route path="/login" element={<Navigate to="/" replace />} />
            <Route path="/register" element={<Navigate to="/" replace />} />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </Suspense>
      </main>
      <AchievementCenter />
    </MeContext>
  );
}

function StudioPassport() {
  const { id = '' } = useParams();
  return canManage(useMe(), id) ? <PassportPage /> : <Navigate to="/" replace />;
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
          <div className="header-account">
            <Link className="button primary" to="/login">
              Sign in
            </Link>
            <Link className="button secondary" to="/register">
              Create account
            </Link>
          </div>
        </div>
      </header>
      <main className="main-content">
        <Suspense fallback={<Loading label="Loading page…" />}>
          <Routes>
            <Route path="/" element={<Landing />} />
            <Route path="/register" element={<AuthPage mode="register" />} />
            <Route path="*" element={<AuthPage mode="login" />} />
          </Routes>
        </Suspense>
      </main>
    </>
  );
}

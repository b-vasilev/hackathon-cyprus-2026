import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, NavLink, Route, Routes } from 'react-router-dom';
import { AchievementCenter } from './components/Achievements';
import { Brand } from './components/ui';
import { installConsoleEgg } from './foundations/consoleEgg';
import { Connections } from './pages/Connections';
import { FoundationAchievements, FoundationModulePage, Foundations } from './pages/Foundations';
import { NotFound } from './pages/NotFound';
import { PassportPage } from './pages/Passport';
import { NewProduct, Products } from './pages/Products';
import '@fontsource/geist/400.css';
import '@fontsource/geist/500.css';
import '@fontsource/geist/600.css';
import '@fontsource/geist-mono/400.css';
import './styles.css';

const root = document.querySelector('#root');
if (!root) {
  throw new Error('Missing root element');
}

installConsoleEgg();

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false, staleTime: 5000 } },
});

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <header className="site-header">
          <div className="header-inner">
            <Brand />
            <nav aria-label="Main navigation">
              <NavLink to="/" end>
                Product spaces
              </NavLink>
              <NavLink to="/foundations">Foundations</NavLink>
              <NavLink to="/connections">Connections</NavLink>
            </nav>
            <span className="header-caption">Know the product. Find your way.</span>
          </div>
        </header>
        <main className="main-content">
          <Routes>
            <Route path="/" element={<Products />} />
            <Route path="/products/new" element={<NewProduct />} />
            <Route path="/products/:id" element={<PassportPage />} />
            <Route path="/foundations" element={<Foundations />} />
            <Route path="/foundations/achievements" element={<FoundationAchievements />} />
            <Route path="/foundations/:id" element={<FoundationModulePage />} />
            <Route path="/connections" element={<Connections />} />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </main>
        <AchievementCenter />
        <footer className="site-footer">
          <span>Wayfinder</span>
          <span>Source-backed drafts. Human understanding.</span>
        </footer>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>
);

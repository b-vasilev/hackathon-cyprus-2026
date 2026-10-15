import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { App } from './App';
import { installConsoleEgg } from './foundations/consoleEgg';
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
        <App />
        <footer className="site-footer">
          <span>Wayfinder</span>
          <span>Source-backed drafts. Human understanding.</span>
        </footer>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>
);

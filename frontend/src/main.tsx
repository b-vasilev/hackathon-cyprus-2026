import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

const root = document.querySelector('#root');
if (!root) {
  throw new Error('Missing root element');
}

// Empty React entry point. Product UI will be implemented separately.
createRoot(root).render(<StrictMode />);

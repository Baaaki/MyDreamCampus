import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router';
import { QueryProvider } from '@/components/providers/query-provider';
import { ThemeProvider } from '@/components/providers/theme-provider';
import { ErrorBoundary } from '@/components/error-boundary';
import { AppRoutes } from './routes';
import './index.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      <BrowserRouter>
        <QueryProvider>
          <ThemeProvider>
            <AppRoutes />
          </ThemeProvider>
        </QueryProvider>
      </BrowserRouter>
    </ErrorBoundary>
  </StrictMode>,
);

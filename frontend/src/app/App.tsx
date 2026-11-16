import { BrowserRouter, Routes, Route, Navigate } from 'react-router';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from '@/shared/api';
import { AppLayout } from '@/app/AppLayout';
import { TraineeRoute } from '@/app/routes';
import { ChatPage } from '@/pages/chat';
import { RoutePage } from '@/pages/route';
import { TicketsPage } from '@/pages/tickets';
import { LoginPage } from '@/pages/login';
import { WikiPage } from '@/pages/wiki/WikiPage';

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<AppLayout />}>
            <Route
              path="/"
              element={
                <TraineeRoute>
                  <ChatPage />
                </TraineeRoute>
              }
            />
            <Route
              path="/chat"
              element={
                <TraineeRoute>
                  <ChatPage />
                </TraineeRoute>
              }
            />
            <Route path="/route" element={<RoutePage />} />
            <Route path="/dashboard" element={<RoutePage />} />
            <Route path="/wiki" element={<WikiPage />} />
            <Route path="/wiki/:documentId" element={<WikiPage />} />
            <Route path="/tickets" element={<TicketsPage />} />
            <Route path="/tickets/:ticketId" element={<TicketsPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export { App };

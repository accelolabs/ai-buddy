import { BrowserRouter, Routes, Route, Navigate } from 'react-router';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from '@/shared/api';
import { AppLayout } from '@/app/AppLayout';
import { TraineeRoute } from '@/app/routes';
import { ChatPage } from '@/pages/chat';
import { DashboardPage } from '@/pages/dashboard';
import { TicketsPage } from '@/pages/tickets';

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
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
            <Route path="/dashboard" element={<DashboardPage />} />
          </Route>
          <Route path="/tickets" element={<TicketsPage />} />
          <Route path="/tickets/:ticketId" element={<TicketsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export { App };

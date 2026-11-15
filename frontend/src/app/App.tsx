import { BrowserRouter, Routes, Route } from 'react-router';
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
            <Route path="/dashboard" element={<DashboardPage />} />
          </Route>
          <Route path="/tickets" element={<TicketsPage />} />
          <Route path="/tickets/:ticketId" element={<TicketsPage />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export { App };

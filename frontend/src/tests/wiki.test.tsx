import { afterEach, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router';
import { headingAnchors } from '@/pages/wiki/anchors';
import { WikiPage } from '@/pages/wiki/WikiPage';
import { queryClient } from '@/shared/api';
import { ChatMessageItem } from '@/widgets/ai-buddy-chat/ui/ChatMessageItem';

afterEach(() => {
  cleanup();
  queryClient.clear();
  vi.unstubAllGlobals();
});
it('matches raw heading slugs, collision suffixes, fenced code and closing hashes', () => {
  const text =
    '# Привет, **мир**! ###\n# Привет, **мир**!\n# привет-мир-1\n# Привет, **мир**!\n```md\n# Ignore\n```\n    # Indented\n## Tab\tspace\n###\n###';
  expect([...headingAnchors(text).values()]).toEqual([
    'привет-мир',
    'привет-мир-1',
    'привет-мир-1-1',
    'привет-мир-2',
    'tabspace',
    '',
    '-1',
  ]);
});
it('renders safe Markdown with backend-compatible duplicate heading IDs', async () => {
  vi.stubGlobal(
    'fetch',
    vi
      .fn()
      .mockResolvedValue(
        new Response(
          '# **Раздел**\n\n## **Раздел**\n\n## [Обзор](https://example.com)\n\nSetext heading\n---\n\n<script>alert(1)</script>\n\n[опасно](javascript:alert)',
        ),
      ),
  );
  const { container } = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/wiki/doc']}>
        <Routes>
          <Route path="/wiki/:documentId" element={<WikiPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  await screen.findByRole('heading', { level: 2, name: 'Раздел' });
  expect(container.querySelector('h1#раздел')).toBeInTheDocument();
  expect(container.querySelector('h2#раздел-1')).toBeInTheDocument();
  expect(container.querySelector('h2#обзорhttpsexamplecom')).toBeInTheDocument();
  expect(container.querySelector('h2#setext-heading')).toBeInTheDocument();
  expect(container.querySelector('script')).toBeNull();
  expect(screen.getByText('опасно').getAttribute('href')).not.toMatch(/^javascript:/);
});
it('links a chat source by document ID and anchor', () => {
  render(
    <MemoryRouter>
      <ChatMessageItem
        busy={false}
        onRetry={() => {}}
        message={{
          id: 'm',
          role: 'assistant',
          content: 'Ответ',
          status: 'completed',
          stages: [],
          source: {
            document_id: 'doc-id',
            anchor: 'раздел-1',
            ordinal: 0,
            title: 'Статья',
            quote: 'Цитата',
          },
        }}
      />
    </MemoryRouter>,
  );
  expect(screen.getByRole('link')).toHaveAttribute(
    'href',
    `/wiki/doc-id#${encodeURIComponent('раздел-1')}`,
  );
});

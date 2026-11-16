import { useEffect } from 'react';
import { ArrowLeft } from 'lucide-react';
import { Link, useLocation, useParams } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import Markdown from 'react-markdown';
import rehypeSlug from 'rehype-slug';
import type { Components } from 'react-markdown';
import { request } from '@/shared/api';
import type { Article } from '@/shared/api/contracts';
import { RequestState } from '@/shared/ui/RequestState';
import { headingAnchors } from './anchors';

export function WikiPage() {
  const { documentId } = useParams();
  const { hash } = useLocation();
  const articles = useQuery({
    queryKey: ['wiki'],
    queryFn: ({ signal }) => request<Article[]>('/api/wiki', { signal }),
    enabled: !documentId,
  });
  const article = useQuery({
    queryKey: ['wiki', documentId],
    queryFn: ({ signal }) =>
      request<string>(`/api/wiki/${encodeURIComponent(documentId!)}`, { signal }),
    enabled: !!documentId,
  });
  useEffect(() => {
    if (!article.data || !hash) return;
    let anchor = hash.slice(1);
    try {
      anchor = decodeURIComponent(anchor);
    } catch {
      /* Keep a malformed fragment as literal text. */
    }
    document.getElementById(anchor)?.scrollIntoView();
  }, [article.data, hash]);
  // rehype-slug supplies heading IDs; align ATX headings with the API's raw-title algorithm.
  const ids = headingAnchors(article.data || '');
  const headings: Components = {};
  for (const tag of ['h1', 'h2', 'h3', 'h4', 'h5', 'h6'] as const) {
    const Tag = tag;
    headings[tag] = ({ node, children, id }) => (
      <Tag id={ids.get(node?.position?.start.line || 0) ?? id}>{children}</Tag>
    );
  }
  return (
    <div className="min-h-full bg-[#F8FAFC] p-4 sm:p-6">
      <div className="mx-auto max-w-4xl">
        <h1 className="mb-6 text-3xl font-bold">База знаний</h1>
        {documentId ? (
          <>
            <Link to="/wiki" className="inline-flex items-center gap-1.5 text-sky-700">
              <ArrowLeft className="h-4 w-4" aria-hidden="true" />
              Все статьи
            </Link>
            <RequestState
              loading={article.isPending}
              error={article.error}
              retry={() => void article.refetch()}
            />
            {article.data !== undefined && (
              <article className="wiki-markdown mt-4 rounded-2xl border border-slate-200 bg-white p-6 sm:p-8">
                <Markdown skipHtml rehypePlugins={[rehypeSlug]} components={headings}>
                  {article.data}
                </Markdown>
              </article>
            )}
          </>
        ) : (
          <>
            <RequestState
              loading={articles.isPending}
              error={articles.error}
              retry={() => void articles.refetch()}
              empty={
                articles.isSuccess && !articles.data.length
                  ? 'Доступных статей пока нет'
                  : undefined
              }
            />
            <div className="grid gap-3">
              {articles.data?.map((item) => (
                <Link
                  key={item.id}
                  to={`/wiki/${encodeURIComponent(item.id)}`}
                  className="rounded-2xl border border-slate-200 bg-white p-5 shadow-xs"
                >
                  <h2 className="font-bold text-sky-800">{item.title}</h2>
                  <p className="text-sm text-slate-500">{item.team}</p>
                </Link>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

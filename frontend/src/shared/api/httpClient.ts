import { changeUser, sessionSignal } from './session';

export class ApiError extends Error {
  readonly status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

export async function fetchApi(endpoint: string, options: RequestInit = {}): Promise<Response> {
  const signal = AbortSignal.any([sessionSignal(), ...(options.signal ? [options.signal] : [])]);
  const headers = new Headers(options.headers);
  if (options.body) headers.set('Content-Type', 'application/json');
  const response = await fetch(`${import.meta.env.VITE_API_URL || ''}${endpoint}`, {
    ...options,
    headers,
    signal,
    credentials: 'include',
  });
  signal.throwIfAborted();
  if (!response.ok) {
    const text = await response.text();
    signal.throwIfAborted();
    let message = text.trim() || `HTTP ${response.status}`;
    try {
      message = JSON.parse(text).error || message;
    } catch {
      /* Plain text errors are part of the API. */
    }
    if (response.status === 401) changeUser(null);
    throw new ApiError(response.status, message);
  }
  return response;
}

export async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const response = await fetchApi(endpoint, options);
  if (response.status === 204) return undefined as T;
  return response.headers.get('Content-Type')?.includes('application/json')
    ? response.json()
    : (response.text() as Promise<T>);
}

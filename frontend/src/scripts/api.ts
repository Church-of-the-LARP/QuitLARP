import createClient from 'openapi-fetch';
import type { paths } from '../api/schema.ts';

export const apiUrl = (
  import.meta.env.VITE_API_URL ?? 'http://localhost:8888'
).replace(/\/$/, '');

// credentials: 'include' so the httpOnly session cookie goes out with every call
const client = createClient<paths>({
  baseUrl: apiUrl,
  credentials: 'include',
});

export const getErrorText = (
  err: { detail?: unknown; status?: unknown } | undefined,
  fallback = 'Something went wrong',
) => {
  if (!err) return fallback;
  if (typeof err.detail === 'string' && err.detail) return err.detail;
  if (typeof err.status === 'number') {
    return `Request failed (HTTP ${err.status})`;
  }

  return fallback;
};

export default client;

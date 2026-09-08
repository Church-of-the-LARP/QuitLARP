/**
 * Typed OpenAPI client (schema regenerated from the backend at container
 * startup) plus small shared helpers.
 */
import createClient from "openapi-fetch";
import type { paths } from "./schema";

export const API_BASE_URL: string =
  ((import.meta.env.VITE_API_URL as string | undefined) ?? "http://localhost:8888").replace(
    /\/$/,
    "",
  );

/**
 * credentials: "include" so the httpOnly session cookie set by the backend
 * (localhost:8888) is sent along with every request.
 */
export const client = createClient<paths>({
  baseUrl: API_BASE_URL,
  credentials: "include",
});

/** Stable error text from a failed API call. */
export function errorMessage(
  err: { detail?: unknown; status?: unknown } | undefined,
  fallback = "Something went wrong",
): string {
  if (!err) {
    return fallback;
  }
  const { detail, status } = err;
  if (typeof detail === "string" && detail.length > 0) {
    return detail;
  }
  return typeof status === "number" ? `Request failed (HTTP ${status})` : fallback;
}

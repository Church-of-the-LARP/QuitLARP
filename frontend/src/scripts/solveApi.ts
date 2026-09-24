// Solve flow transport. Kept as plain fetch with local types because the solve
// endpoints are not part of the generated OpenAPI schema yet.
import { apiUrl } from "./api.ts";

export type SolveChapter = {
  id: number;
  position: number;
  title: string;
  description: string;
  taskFile: string;
  startMode: string;
  timeLimitMinutes: number | null;
};

export type SolveAssessment = {
  id: number;
  title: string;
  description: string;
  kind: string;
  timeLimitMinutes: number | null;
  chapters: SolveChapter[];
};

export type SolveSession = {
  assessmentId: number;
  chapterId: number;
  chapterTitle: string;
  taskFile: string;
  fileName: string;
  content: string;
  deadline: string;
  status: string;
  timeLimitMinutes: number | null;
};

export type SolveResult = {
  name: string;
  passed: boolean;
  detail: string;
};

export type SolveTestRun = {
  results: SolveResult[];
  raw: string;
};

export type SolveSolution = {
  id: number;
  assessmentId: number;
  public: boolean;
  commitSha: string;
  url: string;
  createdAt: string;
};

export class SolveApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "SolveApiError";
    this.status = status;
  }
}

type RequestOptions = {
  method: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
};

const errorMessage = async (response: Response) => {
  let detail: unknown;
  try {
    const data = (await response.json()) as {
      detail?: unknown;
      error?: unknown;
    };
    detail = data?.detail ?? data?.error;
  } catch {
    detail = undefined;
  }
  if (typeof detail === "string" && detail) return detail;
  return `Request failed (HTTP ${response.status})`;
};

async function request<T>(path: string, options: RequestOptions): Promise<T> {
  const response = await fetch(`${apiUrl}${path}`, {
    method: options.method,
    credentials: "include",
    headers:
      options.body === undefined
        ? undefined
        : { "Content-Type": "application/json" },
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });

  if (!response.ok) {
    throw new SolveApiError(response.status, await errorMessage(response));
  }

  const text = await response.text();
  if (!text) return undefined as unknown as T;

  return JSON.parse(text) as T;
}

export async function getAssessment(id: number) {
  const data = await request<{ assessment: SolveAssessment }>(
    `/api/v1/assessments/${id}`,
    { method: "GET" },
  );

  return data.assessment;
}

export async function getCurrentSession(id: number) {
  const data = await request<{ session: SolveSession }>(
    `/api/v1/assessments/${id}/sessions/current`,
    { method: "GET" },
  );

  return data.session;
}

export async function startSession(id: number) {
  const data = await request<{ session: SolveSession }>(
    `/api/v1/assessments/${id}/sessions`,
    { method: "POST" },
  );

  return data.session;
}

export async function saveFile(id: number, content: string) {
  const data = await request<{ session: SolveSession }>(
    `/api/v1/assessments/${id}/sessions/current/file`,
    { method: "PUT", body: { content } },
  );

  return data.session;
}

export async function runTests(id: number) {
  const data = await request<SolveTestRun>(
    `/api/v1/assessments/${id}/sessions/current/tests`,
    { method: "POST" },
  );

  return { results: data.results ?? [], raw: data.raw ?? "" };
}

export async function submitSolution(id: number, isPublic: boolean) {
  const data = await request<{ solution: SolveSolution }>(
    `/api/v1/assessments/${id}/sessions/current/submit`,
    { method: "POST", body: { public: isPublic } },
  );

  return data.solution;
}

export async function abandonSession(id: number) {
  await request<undefined>(`/api/v1/assessments/${id}/sessions/current`, {
    method: "DELETE",
  });
}

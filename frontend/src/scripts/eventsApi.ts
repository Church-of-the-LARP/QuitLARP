// Event, attempt and collaboration transport. The generated OpenAPI schema
// does not cover streaming, so this module talks to the event endpoints
// directly while keeping the same cookie-based session and problem errors.
import { apiUrl } from "./api.ts";

export type EventStatus = "draft" | "live" | "ended";
export type AttemptKind = "solo" | "team";
export type AttemptMemberRole = "owner" | "member";

export type EventDeployment = {
  attemptId: number;
  publicId: string;
  url: string;
  commitSha: string;
  status: "empty" | "deploying" | "live" | "failed";
  error?: string;
  deployedAt?: string;
  published: boolean;
};

export type AttemptMember = {
  userId: number;
  username: string;
  role: AttemptMemberRole;
  joinedAt: string;
};

export type EventAttempt = {
  id: number;
  eventId: number;
  eventTaskId: number;
  name: string;
  kind: AttemptKind;
  owner?: { id: number; username: string };
  members: AttemptMember[];
  repoId: string;
  repoUrl: string;
  joinCode?: string;
  isMine: boolean;
  canEdit: boolean;
  dirty: boolean;
  deployment: EventDeployment;
};

export type EventTask = {
  id: number;
  eventId: number;
  assessmentId: number;
  position: number;
  title: string;
  description: string;
  attempts: EventAttempt[];
};

export type EventMember = {
  userId: number;
  username: string;
  role: "host" | "member";
  joinedAt: string;
};

export type EventSummary = {
  id: number;
  title: string;
  description: string;
  status: EventStatus;
  host?: { id: number; username: string };
  startsAt?: string;
  endsAt?: string;
  taskCount: number;
  createdAt: string;
};

export type EventDetail = EventSummary & {
  isHost: boolean;
  joinCode?: string;
  tasks: EventTask[];
  members: EventMember[];
};

export type UnboundTask = {
  id: number;
  title: string;
  description: string;
  kind: string;
  difficulty: string;
};

export type WorkspaceFile = {
  path: string;
  size: number;
  binary: boolean;
  truncated: boolean;
};

export type CollabDoc = {
  path: string;
  content: string;
  version: number;
  dirty: boolean;
};

export type Workspace = {
  docs: CollabDoc[];
  files: WorkspaceFile[];
  deleted: string[];
  dirty: boolean;
};

export type CollabOp = {
  path: string;
  offset: number;
  deleteCount: number;
  insert: string;
  baseVersion: number;
  version: number;
  clientId: string;
  userId: number;
  username: string;
};

export type CollabMessage =
  | { type: "snapshot"; docs?: CollabDoc[]; dirty?: boolean }
  | { type: "op"; op: CollabOp; dirty?: boolean }
  | { type: "presence"; users?: { id: number; username: string }[] }
  | { type: "committed"; commit: string }
  | { type: "reload" };

export class EventsApiError extends Error {
  status: number;
  constructor(message: string, status = 0) {
    super(message);
    this.name = "EventsApiError";
    this.status = status;
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const res = await fetch(`${apiUrl}${path}`, {
    credentials: "include",
    ...init,
    headers,
  });
  const text = await res.text();
  if (!res.ok) {
    let detail = "";
    try {
      const parsed = JSON.parse(text) as { detail?: unknown };
      if (typeof parsed.detail === "string") detail = parsed.detail;
    } catch {
      // Non-JSON errors fall back to the status line below.
    }
    throw new EventsApiError(
      detail || `Request failed (HTTP ${res.status})`,
      res.status,
    );
  }
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export async function listEvents(): Promise<EventSummary[]> {
  const data = await request<{ events: EventSummary[] }>("/api/v1/events");
  return data.events ?? [];
}

export async function createEvent(input: {
  title: string;
  description: string;
  taskIds: number[];
  startsAt?: string;
  endsAt?: string;
}): Promise<EventDetail> {
  const data = await request<{ event: EventDetail }>("/api/v1/events", {
    method: "POST",
    body: JSON.stringify(input),
  });
  return data.event;
}

export async function getEvent(id: number): Promise<EventDetail> {
  const data = await request<{ event: EventDetail }>(`/api/v1/events/${id}`);
  return data.event;
}

export async function patchEvent(
  id: number,
  input: { title?: string; description?: string; startsAt?: string; endsAt?: string },
): Promise<EventDetail> {
  const data = await request<{ event: EventDetail }>(`/api/v1/events/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
  return data.event;
}

export async function startEvent(id: number): Promise<EventDetail> {
  const data = await request<{ event: EventDetail }>(
    `/api/v1/events/${id}/start`,
    { method: "POST" },
  );
  return data.event;
}

export async function endEvent(id: number): Promise<EventDetail> {
  const data = await request<{ event: EventDetail }>(
    `/api/v1/events/${id}/end`,
    { method: "POST" },
  );
  return data.event;
}

export async function joinEvent(code: string): Promise<EventDetail> {
  const data = await request<{ event: EventDetail }>("/api/v1/events/join", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
  return data.event;
}

export async function joinAttempt(code: string): Promise<EventDetail> {
  const data = await request<{ event: EventDetail }>("/api/v1/attempts/join", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
  return data.event;
}

export async function addEventMember(
  id: number,
  username: string,
): Promise<EventDetail> {
  const data = await request<{ event: EventDetail }>(
    `/api/v1/events/${id}/members`,
    { method: "POST", body: JSON.stringify({ username }) },
  );
  return data.event;
}

export async function removeEventMember(
  id: number,
  userId: number,
): Promise<EventDetail> {
  const data = await request<{ event: EventDetail }>(
    `/api/v1/events/${id}/members/${userId}`,
    { method: "DELETE" },
  );
  return data.event;
}

export async function listUnboundTasks(): Promise<UnboundTask[]> {
  const data = await request<{ assessments: UnboundTask[] }>(
    "/api/v1/assessments?kind=unbound&limit=100",
  );
  return data.assessments ?? [];
}

export async function createAttempt(
  eventId: number,
  taskId: number,
  input: { kind: AttemptKind; name?: string; ownerUsername?: string },
): Promise<EventAttempt> {
  const data = await request<{ attempt: EventAttempt }>(
    `/api/v1/events/${eventId}/tasks/${taskId}/attempts`,
    { method: "POST", body: JSON.stringify(input) },
  );
  return data.attempt;
}

export async function addAttemptMember(
  eventId: number,
  attemptId: number,
  username: string,
): Promise<EventAttempt> {
  const data = await request<{ attempt: EventAttempt }>(
    `/api/v1/events/${eventId}/attempts/${attemptId}/members`,
    { method: "POST", body: JSON.stringify({ username }) },
  );
  return data.attempt;
}

export async function removeAttemptMember(
  eventId: number,
  attemptId: number,
  userId: number,
): Promise<EventAttempt> {
  const data = await request<{ attempt: EventAttempt }>(
    `/api/v1/events/${eventId}/attempts/${attemptId}/members/${userId}`,
    { method: "DELETE" },
  );
  return data.attempt;
}

export async function getWorkspace(
  eventId: number,
  attemptId: number,
): Promise<Workspace> {
  return request(
    `/api/v1/events/${eventId}/attempts/${attemptId}/workspace`,
  );
}

export async function applyEdit(
  eventId: number,
  attemptId: number,
  op: {
    path: string;
    offset: number;
    deleteCount: number;
    insert: string;
    baseVersion: number;
    clientId: string;
  },
): Promise<{ version: number; op: CollabOp }> {
  return request(`/api/v1/events/${eventId}/attempts/${attemptId}/edits`, {
    method: "POST",
    body: JSON.stringify(op),
  });
}

export async function createFile(
  eventId: number,
  attemptId: number,
  path: string,
  content: string,
): Promise<Workspace> {
  return request(`/api/v1/events/${eventId}/attempts/${attemptId}/files`, {
    method: "POST",
    body: JSON.stringify({ path, content }),
  });
}

export async function deleteFile(
  eventId: number,
  attemptId: number,
  path: string,
): Promise<Workspace> {
  return request(
    `/api/v1/events/${eventId}/attempts/${attemptId}/files?path=${encodeURIComponent(path)}`,
    { method: "DELETE" },
  );
}

export async function commitAttempt(
  eventId: number,
  attemptId: number,
  message?: string,
): Promise<EventAttempt> {
  const data = await request<{ attempt: EventAttempt }>(
    `/api/v1/events/${eventId}/attempts/${attemptId}/commit`,
    { method: "POST", body: JSON.stringify({ message }) },
  );
  return data.attempt;
}

export async function deployAttempt(
  eventId: number,
  attemptId: number,
): Promise<EventAttempt> {
  const data = await request<{ attempt: EventAttempt }>(
    `/api/v1/events/${eventId}/attempts/${attemptId}/deploy`,
    { method: "POST" },
  );
  return data.attempt;
}

export async function publishDeployment(
  eventId: number,
  attemptId: number,
  published: boolean,
): Promise<EventAttempt> {
  const data = await request<{ attempt: EventAttempt }>(
    `/api/v1/events/${eventId}/attempts/${attemptId}/deployment`,
    { method: "PATCH", body: JSON.stringify({ published }) },
  );
  return data.attempt;
}

export function openStream(
  eventId: number,
  attemptId: number,
): EventSource {
  return new EventSource(
    `${apiUrl}/api/v1/events/${eventId}/attempts/${attemptId}/stream`,
    { withCredentials: true },
  );
}

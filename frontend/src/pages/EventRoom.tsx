import { useEffect, useMemo, useRef, useState } from "react";
import Editor from "@monaco-editor/react";
import { ArrowLeft, File, FilePlus, Trash2, X } from "lucide-react";
import { Link, useLocation, useParams } from "react-router-dom";
import usePageTitle from "../scripts/usePageTitle.ts";
import useAuth from "../scripts/useAuth.tsx";
import { APP_THEME, defineAppTheme } from "../scripts/monacoTheme.ts";
import {
  addAttemptMember,
  applyEdit,
  commitAttempt,
  createAttempt,
  createFile,
  deleteFile,
  deployAttempt,
  endEvent,
  EventsApiError,
  getEvent,
  getWorkspace,
  openStream,
  publishDeployment,
  startEvent,
  type CollabDoc,
  type CollabMessage,
  type CollabOp,
  type EventAttempt,
  type EventDetail,
  type EventTask,
  type Workspace,
} from "../scripts/eventsApi.ts";

const statusStyles: Record<string, string> = {
  draft: "bg-raised text-muted",
  live: "bg-success/20 text-success",
  ended: "bg-accent-secondary/20 text-accent-secondary",
};

const languageByExt: Record<string, string> = {
  ts: "typescript",
  tsx: "typescript",
  js: "javascript",
  jsx: "javascript",
  mjs: "javascript",
  json: "json",
  html: "html",
  htm: "html",
  css: "css",
  scss: "scss",
  md: "markdown",
  py: "python",
  go: "go",
  rs: "rust",
  svg: "xml",
  txt: "plaintext",
};

const languageOf = (path: string) =>
  languageByExt[path.split(".").pop()?.toLowerCase() ?? ""] ?? "plaintext";

// diff returns the single splice that turns previous into next. Monaco and
// JavaScript indices are UTF-16 code units, which is what the server expects.
const diff = (previous: string, next: string) => {
  let start = 0;
  while (
    start < previous.length &&
    start < next.length &&
    previous[start] === next[start]
  ) {
    start++;
  }
  let prevEnd = previous.length;
  let nextEnd = next.length;
  while (
    prevEnd > start &&
    nextEnd > start &&
    previous[prevEnd - 1] === next[nextEnd - 1]
  ) {
    prevEnd--;
    nextEnd--;
  }
  return {
    offset: start,
    deleteCount: prevEnd - start,
    insert: next.slice(start, nextEnd),
  };
};

const splice = (text: string, op: { offset: number; deleteCount: number; insert: string }) =>
  text.slice(0, op.offset) + op.insert + text.slice(op.offset + op.deleteCount);

const newClientId = () =>
  typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `client-${Math.random().toString(36).slice(2)}`;

const docsToMap = (docs: CollabDoc[]) => {
  const map = new Map<string, CollabDoc>();
  for (const doc of docs) map.set(doc.path, doc);
  return map;
};

export default function EventRoom() {
  const location = useLocation();
  const isInterviewPath = location.pathname.startsWith("/interviews/");
  const backTo = isInterviewPath ? "/dashboard" : "/collaborations";
  const backLabel = isInterviewPath ? "Dashboard" : "Collaborations";
  usePageTitle(isInterviewPath ? "Interview" : "Collaboration");
  const params = useParams();
  const eventId = Number(params.id);
  const { user } = useAuth();

  const [event, setEvent] = useState<EventDetail | null>(null);
  const [activeTaskId, setActiveTaskId] = useState(0);
  const [activeAttemptId, setActiveAttemptId] = useState(0);
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [activePath, setActivePath] = useState("");
  const [openPaths, setOpenPaths] = useState<string[]>([]);
  const [content, setContent] = useState("");
  const [presence, setPresence] = useState<{ id: number; username: string }[]>([]);
  const [dirty, setDirty] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [inviteName, setInviteName] = useState("");

  const docsRef = useRef<Map<string, CollabDoc>>(new Map());
  const activePathRef = useRef("");
  const contentRef = useRef("");
  const pendingRef = useRef(false);
  const pendingPathRef = useRef("");
  const clientIdRef = useRef(newClientId());
  const sendTimerRef = useRef<number | undefined>(undefined);
  const sendNowRef = useRef<() => void>(() => {});
  const handleMessageRef = useRef<(message: CollabMessage) => void>(() => {});
  const attemptIdRef = useRef(0);
  const reloadWorkspaceRef = useRef<(attemptId: number) => void>(() => {});

  const activeTask: EventTask | undefined = event?.tasks.find(
    (task) => task.id === activeTaskId,
  );
  const attempts = (activeTask?.attempts ?? []).filter(
    (attempt) => !isInterviewPath || attempt.isMine,
  );
  const activeAttempt: EventAttempt | undefined = attempts.find(
    (attempt) => attempt.id === activeAttemptId,
  );

  const setActiveFile = (path: string, fileContent: string) => {
    activePathRef.current = path;
    contentRef.current = fileContent;
    setActivePath(path);
    setContent(fileContent);
  };

  const updateAttempt = (updated: EventAttempt) => {
    setEvent((current) => {
      if (!current) return current;
      return {
        ...current,
        tasks: current.tasks.map((task) => ({
          ...task,
          attempts: task.attempts.map((attempt) =>
            attempt.id === updated.id ? updated : attempt,
          ),
        })),
      };
    });
  };

  const loadEvent = async () => {
    try {
      const loaded = await getEvent(eventId);
      setEvent(loaded);
      setActiveTaskId((current) => {
        if (current && loaded.tasks.some((task) => task.id === current)) {
          return current;
        }
        return loaded.tasks[0]?.id ?? 0;
      });
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load the session");
    }
  };

  const reloadWorkspace = async (attemptId: number) => {
    try {
      const loaded = await getWorkspace(eventId, attemptId);
      setWorkspace(loaded);
      docsRef.current = docsToMap(loaded.docs);
      setDirty(loaded.dirty);
      const active = activePathRef.current;
      const doc = active ? docsRef.current.get(active) : undefined;
      if (doc) {
        setActiveFile(doc.path, doc.content);
      } else if (active) {
        setActiveFile("", "");
      }
      setNotice("");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not load the workspace");
    }
  };
  reloadWorkspaceRef.current = reloadWorkspace;

  useEffect(() => {
    if (Number.isInteger(eventId) && eventId > 0) void loadEvent();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [eventId]);

  useEffect(() => {
    if (!activeTask) {
      setActiveAttemptId(0);
      return;
    }
    setActiveAttemptId((current) => {
      const visible = (activeTask?.attempts ?? []).filter(
        (attempt) => !isInterviewPath || attempt.isMine,
      );
      return current && visible.some((attempt) => attempt.id === current)
        ? current
        : visible[0]?.id ?? 0;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTaskId, event, isInterviewPath]);

  useEffect(() => {
    attemptIdRef.current = activeAttemptId;
    docsRef.current.clear();
    pendingRef.current = false;
    pendingPathRef.current = "";
    setWorkspace(null);
    setDirty(false);
    setActiveFile("", "");
    setOpenPaths([]);
    setPresence([]);
    if (activeAttemptId) void reloadWorkspace(activeAttemptId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeAttemptId]);

  const resyncFile = async (path: string) => {
    const attemptId = attemptIdRef.current;
    if (!attemptId) return;
    try {
      const loaded = await getWorkspace(eventId, attemptId);
      setWorkspace(loaded);
      docsRef.current = docsToMap(loaded.docs);
      setDirty(loaded.dirty);
      const doc = docsRef.current.get(path);
      if (doc) {
        pendingRef.current = false;
        pendingPathRef.current = "";
        setActiveFile(path, doc.content);
      }
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not reload the workspace");
    }
  };

  const handleMessage = (message: CollabMessage) => {
    if (message.type === "presence") {
      setPresence(message.users ?? []);
      return;
    }
    if (message.type === "committed") {
      setDirty(false);
      if (activeAttemptId) void reloadWorkspaceRef.current(activeAttemptId);
      return;
    }
    if (message.type === "reload") {
      if (activeAttemptId) void reloadWorkspaceRef.current(activeAttemptId);
      return;
    }
    if (message.type === "snapshot") {
      const docs = message.docs ?? [];
      docsRef.current = docsToMap(docs);
      setWorkspace((current) =>
        current ? { ...current, docs, dirty: message.dirty ?? current.dirty } : current,
      );
      setDirty(message.dirty ?? false);
      const active = activePathRef.current;
      const doc = docsRef.current.get(active);
      if (doc && !pendingRef.current) setActiveFile(active, doc.content);
      return;
    }
    if (message.type !== "op") return;

    const op: CollabOp = message.op;
    const known = docsRef.current.get(op.path);
    if (known && op.version <= known.version) return;
    const serverContent = known ? splice(known.content, op) : null;
    if (serverContent !== null) {
      docsRef.current.set(op.path, {
        path: op.path,
        content: serverContent,
        version: op.version,
        dirty: true,
      });
      setWorkspace((current) =>
        current
          ? {
              ...current,
              docs: current.docs.map((doc) =>
                doc.path === op.path
                  ? { ...doc, content: serverContent, version: op.version, dirty: true }
                  : doc,
              ),
              dirty: true,
            }
          : current,
      );
    }
    setDirty(true);
    if (op.path !== activePathRef.current) return;

    const mine = op.clientId === clientIdRef.current;
    if (mine) {
      if (op.path === pendingPathRef.current) {
        pendingRef.current = false;
        pendingPathRef.current = "";
      }
      void Promise.resolve().then(() => sendNowRef.current());
      return;
    }
    if (serverContent === null) return;
    if (pendingRef.current) {
      if (op.offset + op.deleteCount <= contentRef.current.length) {
        const merged = splice(contentRef.current, op);
        contentRef.current = merged;
        setContent(merged);
      } else {
        setActiveFile(op.path, serverContent);
      }
      return;
    }
    setActiveFile(op.path, serverContent);
  };
  handleMessageRef.current = handleMessage;

  useEffect(() => {
    if (!activeAttemptId || !Number.isInteger(eventId)) return;
    const source = openStream(eventId, activeAttemptId);
    const types = ["snapshot", "op", "presence", "committed", "reload"] as const;
    const listeners = types.map((type) => {
      const listener = (raw: MessageEvent) => {
        try {
          handleMessageRef.current(JSON.parse(raw.data) as CollabMessage);
        } catch {
          // Ignore malformed frames; the next snapshot repairs state.
        }
      };
      source.addEventListener(type, listener);
      return { type, listener };
    });
    source.onerror = () => setPresence([]);
    return () => {
      for (const { type, listener } of listeners) {
        source.removeEventListener(type, listener);
      }
      source.close();
    };
  }, [eventId, activeAttemptId]);

  const sendNow = () => {
    const path = activePathRef.current;
    const attemptId = attemptIdRef.current;
    if (!path || !attemptId) return;
    if (pendingRef.current) return;
    const doc = docsRef.current.get(path);
    if (!doc) return;
    const local = contentRef.current;
    if (local === doc.content) return;
    const change = diff(doc.content, local);
    pendingRef.current = true;
    pendingPathRef.current = path;
    applyEdit(eventId, attemptId, {
      path,
      ...change,
      baseVersion: doc.version,
      clientId: clientIdRef.current,
    })
      .then(() => setDirty(true))
      .catch((err: unknown) => {
        pendingRef.current = false;
        if (err instanceof EventsApiError && err.status === 409) {
          setNotice("The file changed elsewhere; reloading it.");
          void resyncFile(path);
        } else {
          setNotice(err instanceof Error ? err.message : "Could not save the edit");
        }
      });
  };
  sendNowRef.current = sendNow;

  const scheduleSend = () => {
    if (sendTimerRef.current !== undefined) {
      window.clearTimeout(sendTimerRef.current);
    }
    sendTimerRef.current = window.setTimeout(() => {
      sendTimerRef.current = undefined;
      sendNowRef.current();
    }, 120);
  };

  const visiblePaths = useMemo(() => {
    const set = new Set<string>();
    for (const doc of workspace?.docs ?? []) set.add(doc.path);
    for (const file of workspace?.files ?? []) set.add(file.path);
    for (const path of workspace?.deleted ?? []) set.delete(path);
    return [...set].sort();
  }, [workspace]);

  const binaryPaths = useMemo(
    () => new Set((workspace?.files ?? []).map((file) => file.path)),
    [workspace],
  );

  const openFile = (path: string) => {
    if (binaryPaths.has(path)) {
      setNotice("This file is binary or too large to edit.");
      return;
    }
    const doc = docsRef.current.get(path);
    if (doc) {
      setOpenPaths((current) =>
        current.includes(doc.path) ? current : [...current, doc.path],
      );
      setActiveFile(doc.path, doc.content);
    }
  };

  const closeTab = (path: string) => {
    const index = openPaths.indexOf(path);
    const next = openPaths.filter((entry) => entry !== path);
    setOpenPaths(next);
    if (activePath !== path) return;
    const fallback = next[Math.min(index, next.length - 1)];
    const doc = fallback ? docsRef.current.get(fallback) : undefined;
    if (doc) {
      setActiveFile(doc.path, doc.content);
    } else {
      setActiveFile("", "");
    }
  };

  const handleCreateFile = async () => {
    if (!activeAttemptId) return;
    const name = window.prompt("New file path (for example src/app.js)");
    if (!name?.trim()) return;
    try {
      const loaded = await createFile(eventId, activeAttemptId, name.trim(), "");
      setWorkspace(loaded);
      docsRef.current = docsToMap(loaded.docs);
      setDirty(loaded.dirty);
      const doc = docsRef.current.get(name.trim());
      if (doc) {
        setOpenPaths((current) =>
          current.includes(doc.path) ? current : [...current, doc.path],
        );
        setActiveFile(doc.path, doc.content);
      }
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not create the file");
    }
  };

  const handleDeleteFile = async (path: string) => {
    if (!activeAttemptId || !window.confirm(`Remove ${path} from the workspace?`)) return;
    try {
      const loaded = await deleteFile(eventId, activeAttemptId, path);
      setWorkspace(loaded);
      docsRef.current = docsToMap(loaded.docs);
      setDirty(loaded.dirty);
      setOpenPaths((current) => current.filter((entry) => entry !== path));
      if (activePathRef.current === path) setActiveFile("", "");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not delete the file");
    }
  };

  const handleStartWorkspace = async () => {
    if (!activeTask || busy) return;
    setBusy(true);
    try {
      const created = await createAttempt(eventId, activeTask.id, {
        kind: "solo",
        name: user?.username,
      });
      setActiveAttemptId(created.id);
      await loadEvent();
      setNotice("Workspace ready.");
    } catch (err) {
      setNotice(
        err instanceof Error ? err.message : "Could not start the workspace",
      );
    } finally {
      setBusy(false);
    }
  };

  const handleInvite = async (submit: React.FormEvent<HTMLFormElement>) => {
    submit.preventDefault();
    if (!activeAttempt || !inviteName.trim()) return;
    try {
      const updated = await addAttemptMember(
        eventId,
        activeAttempt.id,
        inviteName.trim(),
      );
      updateAttempt(updated);
      setInviteName("");
      setNotice("Collaborator invited.");
    } catch (err) {
      setNotice(
        err instanceof Error ? err.message : "Could not invite the collaborator",
      );
    }
  };

  const handleCommit = async () => {
    if (!activeAttempt) return;
    setBusy(true);
    try {
      const updated = await commitAttempt(eventId, activeAttempt.id);
      updateAttempt(updated);
      await reloadWorkspace(activeAttempt.id);
      setDirty(false);
      setNotice(
        updated.deployment.status === "live"
          ? `Committed ${updated.deployment.commitSha.slice(0, 8)} and deployed.`
          : "Committed to main. The deployment did not update — see the deployment panel.",
      );
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not commit");
    } finally {
      setBusy(false);
    }
  };

  const handleStart = async () => {
    setBusy(true);
    try {
      setEvent(await startEvent(eventId));
      setNotice("Session is live. Participants can create attempts.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not start the session");
    } finally {
      setBusy(false);
    }
  };

  const handleEnd = async () => {
    if (!event) return;
    const dirtyAttempts = event.tasks
      .flatMap((task) => task.attempts)
      .filter((attempt) => attempt.dirty);
    if (
      dirtyAttempts.length > 0 &&
      !window.confirm(
        `${dirtyAttempts.length} attempt(s) have uncommitted editor changes. Those changes will not be deployed. End the session anyway?`,
      )
    ) {
      return;
    }
    setBusy(true);
    try {
      setEvent(await endEvent(eventId));
      setNotice("Session ended. Attempts are read-only.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not end the session");
    } finally {
      setBusy(false);
    }
  };

  const handleRedeploy = async () => {
    if (!activeAttempt) return;
    setBusy(true);
    try {
      const updated = await deployAttempt(eventId, activeAttempt.id);
      updateAttempt(updated);
      setNotice("Redeployed the current main commit.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Deployment failed");
    } finally {
      setBusy(false);
    }
  };

  const handlePublish = async () => {
    if (!activeAttempt) return;
    setBusy(true);
    try {
      const updated = await publishDeployment(
        eventId,
        activeAttempt.id,
        !activeAttempt.deployment.published,
      );
      updateAttempt(updated);
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not update the deployment");
    } finally {
      setBusy(false);
    }
  };

  const copy = async (value: string) => {
    try {
      await navigator.clipboard.writeText(value);
      setNotice("Copied.");
    } catch {
      setNotice("Copy failed; select the text instead.");
    }
  };

  if (error) {
    return (
      <main className="mx-auto w-[calc(100%-2rem)] py-10 lg:w-[75vw]">
        <div className="rounded-xl bg-surface p-6 text-danger">{error}</div>
        <Link to={backTo} className="mt-4 inline-block text-sm font-semibold text-accent-light">
          Back to {backLabel}
        </Link>
      </main>
    );
  }
  if (!event) {
    return <main className="mx-auto w-[calc(100%-2rem)] py-10 text-muted">Loading…</main>;
  }

  const readOnly = !activeAttempt?.canEdit;
  const repoUrl = activeAttempt?.repoUrl;

  return (
    <main className="mx-auto flex w-[calc(100%-2rem)] flex-col gap-5 py-6 lg:w-[95vw] 2xl:w-[80vw]">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          {!event.solo && (
            <Link
              to={backTo}
              className="inline-flex items-center gap-1 text-xs font-semibold uppercase text-accent-light"
            >
              <ArrowLeft aria-hidden="true" className="h-3.5 w-3.5" />
              {backLabel}
            </Link>
          )}
          <div className="mt-1 flex flex-wrap items-center gap-3">
            <h1 className="text-3xl font-bold text-ink">{event.title}</h1>
            <span
              className={`rounded-full px-2 py-0.5 text-xs font-semibold uppercase ${statusStyles[event.status] ?? "bg-raised text-muted"}`}
            >
              {event.status}
            </span>
            {event.isHost && event.joinCode && !event.solo && !isInterviewPath && (
              <button
                onClick={() => copy(event.joinCode!)}
                className="rounded-full bg-panel px-3 py-1 font-mono text-xs text-muted hover:text-ink"
                title="Copy session join code"
              >
                code: {event.joinCode}
              </button>
            )}
          </div>
          {event.description && <p className="mt-2 max-w-3xl text-sm text-muted">{event.description}</p>}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {event.isHost && event.status === "draft" && !isInterviewPath && (
            <button
              onClick={handleStart}
              disabled={busy}
              className="rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-shell hover:bg-accent-light disabled:opacity-60"
            >
              Start session
            </button>
          )}
          {event.isHost && event.status === "live" && !event.solo && !isInterviewPath && (
            <button
              onClick={handleEnd}
              disabled={busy}
              className="rounded-xl border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-muted hover:text-ink disabled:opacity-60"
            >
              End session
            </button>
          )}
          {activeAttempt && event.isHost && !isInterviewPath && (
            <button
              onClick={handleRedeploy}
              disabled={busy}
              className="rounded-xl border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-muted hover:text-ink disabled:opacity-60"
            >
              Redeploy main
            </button>
          )}
          {activeAttempt && event.isHost && event.status === "ended" && !isInterviewPath && (
            <button
              onClick={handlePublish}
              disabled={busy}
              className={`rounded-xl px-4 py-2 text-sm font-semibold ${
                activeAttempt.deployment.published
                  ? "bg-success/20 text-success"
                  : "bg-accent text-shell hover:bg-accent-light"
              }`}
            >
              {activeAttempt.deployment.published ? "Published" : "Publish deployment"}
            </button>
          )}
        </div>
      </header>

      {notice && (
        <div className="rounded-xl bg-panel px-4 py-2 text-sm text-muted">{notice}</div>
      )}

      {event.tasks.length === 0 ? (
        <div className="rounded-xl bg-surface p-6 text-muted">
          This session has no tasks attached.
        </div>
      ) : (
        <>
          {event.tasks.length > 1 && (
            <div className="flex flex-wrap items-center gap-2">
              {event.tasks.map((task) => (
                <button
                  key={task.id}
                  onClick={() => setActiveTaskId(task.id)}
                  className={`rounded-xl px-4 py-2 text-sm font-semibold transition ${
                    task.id === activeTaskId
                      ? "bg-accent text-shell"
                      : "bg-surface text-muted hover:text-ink"
                  }`}
                >
                  {task.title}
                </button>
              ))}
            </div>
          )}

          {event.status === "draft" ? (
            <div className="rounded-xl bg-surface p-6 text-muted">
              The host must start the session before workspaces can be opened.
            </div>
          ) : (
            <>
              {activeTask &&
                activeTask.kind === "unbound" &&
                !activeAttempt && (
                  <section className="rounded-xl bg-surface p-5">
                    {event.status === "live" ? (
                      <>
                        <p className="text-sm text-muted">
                          Start your workspace for this task.
                        </p>
                        <button
                          onClick={() => void handleStartWorkspace()}
                          disabled={busy}
                          className="mt-3 rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-shell hover:bg-accent-light disabled:opacity-60"
                        >
                          {busy ? "Starting…" : "Start workspace"}
                        </button>
                      </>
                    ) : (
                      <p className="text-sm text-muted">
                        No workspace is available for this session.
                      </p>
                    )}
                  </section>
                )}

              {activeTask?.kind === "leetcode" && (
                <section className="rounded-xl bg-surface p-5">
                  <div className="flex flex-wrap items-start justify-between gap-4">
                    <div className="min-w-0">
                      <p className="text-xs font-semibold uppercase text-dim">
                        Coding task
                      </p>
                      <h2 className="mt-1 text-lg font-semibold text-ink">
                        {activeTask.title}
                      </h2>
                      <p className="mt-2 line-clamp-3 max-w-2xl text-sm text-muted">
                        {activeTask.description}
                      </p>
                    </div>
                    <Link
                      to={`/assessments/${activeTask.assessmentId}/preview`}
                      className="rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-shell hover:bg-accent-light"
                    >
                      Open assessment
                    </Link>
                  </div>
                </section>
              )}

              {activeAttempt && (
                <div className="grid min-h-[70vh] gap-2 rounded-xl bg-panel p-4 xl:grid-cols-[240px_minmax(0,1fr)]">
                  <aside className="flex min-h-0 flex-col overflow-hidden rounded-xl bg-surface p-3">
                    <div className="mb-2 flex items-center justify-between gap-2 px-1">
                      <p className="text-xs font-semibold uppercase tracking-wide text-dim">
                        Explorer
                      </p>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={handleCreateFile}
                          disabled={readOnly}
                          title="New file"
                          className="rounded p-1 text-muted hover:bg-raised/60 hover:text-ink disabled:opacity-40"
                        >
                          <FilePlus aria-hidden="true" className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => activePath && handleDeleteFile(activePath)}
                          disabled={readOnly || !activePath}
                          title="Delete active file"
                          className="rounded p-1 text-muted hover:bg-raised/60 hover:text-danger disabled:opacity-40"
                        >
                          <Trash2 aria-hidden="true" className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                    <div className="mb-2 px-1">
                      <p className="truncate text-xs text-muted">
                        {activeAttempt.name}
                      </p>
                      <p className="text-[11px] uppercase tracking-wide text-dim">
                        {activeAttempt.kind === "team" ? "team" : "solo"} workspace
                      </p>
                    </div>
                    <ul className="flex min-h-0 flex-1 flex-col overflow-auto font-mono text-[13px]">
                      {visiblePaths.map((path) => {
                        const segments = path.split("/");
                        const name = segments[segments.length - 1];
                        return (
                          <li key={path} className="group flex items-center gap-1">
                            <button
                              onClick={() => openFile(path)}
                              title={path}
                              style={{ paddingLeft: `${8 + (segments.length - 1) * 12}px` }}
                              className={`flex min-w-0 flex-1 items-center gap-2 rounded py-1 pr-1 text-left transition hover:bg-raised/40 ${
                                activePath === path ? "bg-raised/40 text-ink" : "text-muted"
                              } ${binaryPaths.has(path) ? "opacity-60" : ""}`}
                            >
                              <File aria-hidden="true" className="h-3.5 w-3.5 shrink-0 text-dim" />
                              <span className="truncate">{name}</span>
                              {docsRef.current.get(path)?.dirty && (
                                <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-accent-light" />
                              )}
                            </button>
                            <button
                              onClick={() => handleDeleteFile(path)}
                              disabled={readOnly}
                              className="hidden rounded p-0.5 text-dim hover:text-danger group-hover:block disabled:opacity-40"
                              title="Remove file"
                            >
                              <X aria-hidden="true" className="h-3.5 w-3.5" />
                            </button>
                          </li>
                        );
                      })}
                      {visiblePaths.length === 0 && (
                        <li className="px-2 py-1 text-xs text-dim">No files</li>
                      )}
                    </ul>
                    <div className="mt-2 border-t border-line pt-2">
                      <p className="text-xs font-semibold uppercase tracking-wide text-dim">
                        In the room
                      </p>
                      <ul className="mt-1 flex flex-wrap gap-1">
                        {presence.map((member) => (
                          <li
                            key={member.id}
                            className="rounded-full bg-raised px-2 py-0.5 text-xs text-muted"
                          >
                            {member.username}
                          </li>
                        ))}
                        {presence.length === 0 && (
                          <li className="text-xs text-dim">Only you</li>
                        )}
                      </ul>
                      {event.monitored &&
                        event.allowInvites &&
                        activeAttempt &&
                        (event.isHost || activeAttempt.owner?.id === user?.id) && (
                          <form onSubmit={handleInvite} className="mt-2 flex gap-1">
                            <input
                              value={inviteName}
                              onChange={(event) => setInviteName(event.target.value)}
                              placeholder="invite username"
                              className="min-w-0 flex-1 rounded-lg bg-panel px-2 py-1 text-xs text-ink outline-none placeholder:text-dim focus:ring-1 focus:ring-accent/40"
                            />
                            <button
                              type="submit"
                              className="rounded-lg bg-accent px-2 py-1 text-xs font-semibold text-shell hover:bg-accent-light"
                            >
                              Invite
                            </button>
                          </form>
                        )}
                    </div>
                  </aside>

                  <section className="flex min-h-0 flex-col overflow-hidden rounded-xl bg-surface">
                    <div className="flex items-center justify-between gap-2 border-b border-line px-2">
                      <div className="flex min-w-0 flex-1 overflow-x-auto [scrollbar-width:none]">
                        {openPaths.length === 0 ? (
                          <span className="px-2 py-2 text-xs text-dim">
                            Open a file from the explorer
                          </span>
                        ) : (
                          openPaths.map((path) => {
                            const name = path.split("/").pop() ?? path;
                            const isDirty = docsRef.current.get(path)?.dirty;
                            return (
                              <div
                                key={path}
                                role="button"
                                tabIndex={0}
                                title={path}
                                onClick={() => openFile(path)}
                                onKeyDown={(event) => {
                                  if (event.key === "Enter" || event.key === " ") {
                                    event.preventDefault();
                                    openFile(path);
                                  }
                                }}
                                className={`flex shrink-0 cursor-pointer items-center gap-2 whitespace-nowrap border-b-2 px-3 py-2 text-[13px] transition ${
                                  activePath === path
                                    ? "border-accent text-ink"
                                    : "border-line text-muted hover:border-line-hover hover:text-ink"
                                }`}
                              >
                                <span>{name}</span>
                                {isDirty && (
                                  <span className="h-2 w-2 rounded-full bg-accent-light" />
                                )}
                                <button
                                  onClick={(event) => {
                                    event.stopPropagation();
                                    closeTab(path);
                                  }}
                                  title="Close"
                                  className="rounded px-1 text-dim hover:bg-raised hover:text-ink"
                                >
                                  <X aria-hidden="true" className="h-3 w-3" />
                                </button>
                              </div>
                            );
                          })
                        )}
                      </div>
                      <div className="flex shrink-0 items-center gap-2 py-1">
                        {activePath && (
                          <>
                            <span className="rounded-md border border-accent-secondary/40 bg-accent-secondary/15 px-2 py-0.5 font-mono text-[11px] text-accent-secondary-hover">
                              {languageOf(activePath)}
                            </span>
                            <span className="hidden rounded-md border border-line bg-surface px-2 py-0.5 font-mono text-[11px] text-dim sm:inline">
                              {activePath.split("/").pop()}
                            </span>
                          </>
                        )}
                      </div>
                    </div>
                    <div className="relative min-h-[55vh] flex-1 overflow-hidden">
                      <Editor
                        height="100%"
                        language={languageOf(activePath)}
                        value={content}
                        onChange={(value) => {
                          const next = value ?? "";
                          if (next === contentRef.current) return;
                          contentRef.current = next;
                          setContent(next);
                          if (!readOnly && activePath) {
                            setDirty(true);
                            scheduleSend();
                          }
                        }}
                        theme={APP_THEME}
                        beforeMount={defineAppTheme}
                        options={{
                          automaticLayout: true,
                          readOnly: readOnly || !activePath,
                          padding: { top: 8, bottom: 8 },
                          minimap: { enabled: false },
                          fontSize: 13,
                        }}
                        className="overflow-hidden"
                      />
                      {!activePath && (
                        <div className="pointer-events-none absolute inset-0 flex items-center justify-center px-4 text-center text-sm text-dim">
                          Open a file from the explorer
                        </div>
                      )}
                    </div>
                    <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line px-3 py-2">
                      <div className="flex min-w-0 flex-wrap items-center gap-2 text-xs text-dim">
                        <span>
                          {readOnly
                            ? "read-only"
                            : dirty
                              ? "uncommitted changes"
                              : "clean"}
                        </span>
                        <span>·</span>
                        <span className="truncate">
                          {activeAttempt.deployment.status === "live"
                            ? `main @ ${activeAttempt.deployment.commitSha.slice(0, 8)}`
                            : activeAttempt.deployment.status}
                        </span>
                        {activeAttempt.deployment.status === "failed" && (
                          <span
                            className="max-w-md truncate text-danger"
                            title={activeAttempt.deployment.error}
                          >
                            {activeAttempt.deployment.error}
                          </span>
                        )}
                        {activeAttempt.deployment.url && (
                          <a
                            href={activeAttempt.deployment.url}
                            target="_blank"
                            rel="noreferrer"
                            className="font-semibold text-accent-light hover:text-accent-lighter"
                          >
                            Open deployment
                          </a>
                        )}
                      </div>
                      {activeAttempt.canEdit && (
                        <button
                          onClick={handleCommit}
                          disabled={busy || !dirty}
                          className="rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-shell hover:bg-accent-light disabled:opacity-50"
                        >
                          {busy ? "Committing…" : "Commit & deploy"}
                        </button>
                      )}
                    </div>
                  </section>

                  {repoUrl && (
                    <button
                      onClick={() => copy(repoUrl)}
                      className="col-span-full truncate text-left font-mono text-xs text-dim hover:text-ink"
                      title="Copy clone URL"
                    >
                      {repoUrl}
                    </button>
                  )}
                </div>
              )}
            </>
          )}
        </>
      )}
    </main>
  );
}

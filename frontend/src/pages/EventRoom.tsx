import { useEffect, useMemo, useRef, useState } from "react";
import Editor from "@monaco-editor/react";
import { Link, useParams } from "react-router-dom";
import usePageTitle from "../scripts/usePageTitle.ts";
import useAuth from "../scripts/useAuth.tsx";
import { APP_THEME, defineAppTheme } from "../scripts/monacoTheme.ts";
import {
  addAttemptMember,
  addEventMember,
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
  joinAttempt,
  openStream,
  publishDeployment,
  removeAttemptMember,
  removeEventMember,
  startEvent,
  type AttemptKind,
  type CollabDoc,
  type CollabMessage,
  type CollabOp,
  type EventAttempt,
  type EventDetail,
  type EventMember,
  type EventTask,
  type Workspace,
} from "../scripts/eventsApi.ts";

const inputClass =
  "w-full rounded-xl border border-line-hover bg-panel px-3 py-2 text-ink outline-none placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30";

const statusStyles: Record<string, string> = {
  draft: "bg-raised text-muted",
  live: "bg-success/20 text-success",
  ended: "bg-accent-secondary/20 text-accent-secondary",
};

const deploymentStyles: Record<string, string> = {
  empty: "bg-raised text-dim",
  deploying: "bg-warning/20 text-warning",
  live: "bg-success/20 text-success",
  failed: "bg-danger/20 text-danger",
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
  usePageTitle("Event");
  const params = useParams();
  const eventId = Number(params.id);
  const { user } = useAuth();

  const [event, setEvent] = useState<EventDetail | null>(null);
  const [activeTaskId, setActiveTaskId] = useState(0);
  const [activeAttemptId, setActiveAttemptId] = useState(0);
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [activePath, setActivePath] = useState("");
  const [content, setContent] = useState("");
  const [presence, setPresence] = useState<{ id: number; username: string }[]>([]);
  const [dirty, setDirty] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const [attemptName, setAttemptName] = useState("");
  const [attemptKind, setAttemptKind] = useState<AttemptKind>("solo");
  const [ownerUsername, setOwnerUsername] = useState("");
  const [teamCode, setTeamCode] = useState("");
  const [memberName, setMemberName] = useState("");
  const [eventMemberName, setEventMemberName] = useState("");

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
  const attempts = activeTask?.attempts ?? [];
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
      setError(err instanceof Error ? err.message : "Could not load the event");
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
    setActiveAttemptId((current) =>
      current && activeTask.attempts.some((attempt) => attempt.id === current)
        ? current
        : activeTask.attempts[0]?.id ?? 0,
    );
  }, [activeTaskId, event]);

  useEffect(() => {
    attemptIdRef.current = activeAttemptId;
    docsRef.current.clear();
    pendingRef.current = false;
    pendingPathRef.current = "";
    setWorkspace(null);
    setDirty(false);
    setActiveFile("", "");
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
      setActiveFile(doc.path, doc.content);
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
      if (doc) setActiveFile(doc.path, doc.content);
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
      if (activePathRef.current === path) setActiveFile("", "");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not delete the file");
    }
  };

  const handleCreateAttempt = async (submit: React.FormEvent<HTMLFormElement>) => {
    submit.preventDefault();
    if (!activeTask) return;
    setBusy(true);
    try {
      const created = await createAttempt(eventId, activeTask.id, {
        kind: attemptKind,
        name: attemptName.trim() || undefined,
        ownerUsername: ownerUsername.trim() || undefined,
      });
      setActiveAttemptId(created.id);
      await loadEvent();
      setAttemptName("");
      setOwnerUsername("");
      setNotice("Attempt created and deployed from its initial commit.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not create the attempt");
    } finally {
      setBusy(false);
    }
  };

  const handleJoinTeam = async (submit: React.FormEvent<HTMLFormElement>) => {
    submit.preventDefault();
    if (!teamCode.trim()) return;
    setBusy(true);
    try {
      const joined = await joinAttempt(teamCode.trim());
      setEvent(joined);
      const joinedAttempt = joined.tasks
        .flatMap((task) => task.attempts)
        .find(
          (attempt) =>
            attempt.isMine &&
            attempt.joinCode?.toUpperCase() === teamCode.trim().toUpperCase(),
        );
      if (joinedAttempt) {
        setActiveTaskId(joinedAttempt.eventTaskId);
        setActiveAttemptId(joinedAttempt.id);
      }
      setTeamCode("");
      setNotice("Joined the team attempt.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not join the team");
    } finally {
      setBusy(false);
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
      setNotice("Event is live. Participants can create attempts.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not start the event");
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
        `${dirtyAttempts.length} attempt(s) have uncommitted editor changes. Those changes will not be deployed. End the event anyway?`,
      )
    ) {
      return;
    }
    setBusy(true);
    try {
      setEvent(await endEvent(eventId));
      setNotice("Event ended. Attempts are read-only.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not end the event");
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

  const handleAddAttemptMember = async (submit: React.FormEvent<HTMLFormElement>) => {
    submit.preventDefault();
    if (!activeAttempt || !memberName.trim()) return;
    try {
      const updated = await addAttemptMember(
        eventId,
        activeAttempt.id,
        memberName.trim(),
      );
      updateAttempt(updated);
      setMemberName("");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not add the member");
    }
  };

  const handleRemoveAttemptMember = async (member: { userId: number }) => {
    if (!activeAttempt) return;
    try {
      updateAttempt(await removeAttemptMember(eventId, activeAttempt.id, member.userId));
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not remove the member");
    }
  };

  const handleAddEventMember = async (submit: React.FormEvent<HTMLFormElement>) => {
    submit.preventDefault();
    if (!eventMemberName.trim()) return;
    try {
      setEvent(await addEventMember(eventId, eventMemberName.trim()));
      setEventMemberName("");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not add the member");
    }
  };

  const handleRemoveEventMember = async (member: EventMember) => {
    try {
      setEvent(await removeEventMember(eventId, member.userId));
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Could not remove the member");
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
        <Link to="/events" className="mt-4 inline-block text-sm font-semibold text-accent-light">
          Back to events
        </Link>
      </main>
    );
  }
  if (!event) {
    return <main className="mx-auto w-[calc(100%-2rem)] py-10 text-muted">Loading…</main>;
  }

  const readOnly = !activeAttempt?.canEdit;
  const canManageAttempt =
    event.isHost ||
    activeAttempt?.members.some(
      (member) => member.userId === user?.id && member.role === "owner",
    );
  const repoUrl = activeAttempt?.repoUrl;

  return (
    <main className="mx-auto flex w-[calc(100%-2rem)] flex-col gap-5 py-6 lg:w-[95vw] 2xl:w-[80vw]">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <Link to="/events" className="text-xs font-semibold uppercase text-accent-light">
            ← Events
          </Link>
          <div className="mt-1 flex flex-wrap items-center gap-3">
            <h1 className="text-3xl font-bold text-ink">{event.title}</h1>
            <span
              className={`rounded-full px-2 py-0.5 text-xs font-semibold uppercase ${statusStyles[event.status] ?? "bg-raised text-muted"}`}
            >
              {event.status}
            </span>
            {event.isHost && event.joinCode && (
              <button
                onClick={() => copy(event.joinCode!)}
                className="rounded-full bg-panel px-3 py-1 font-mono text-xs text-muted hover:text-ink"
                title="Copy event join code"
              >
                code: {event.joinCode}
              </button>
            )}
          </div>
          {event.description && <p className="mt-2 max-w-3xl text-sm text-muted">{event.description}</p>}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {event.isHost && event.status === "draft" && (
            <button
              onClick={handleStart}
              disabled={busy}
              className="rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-shell hover:bg-accent-light disabled:opacity-60"
            >
              Start event
            </button>
          )}
          {event.isHost && event.status === "live" && (
            <button
              onClick={handleEnd}
              disabled={busy}
              className="rounded-xl border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-muted hover:text-ink disabled:opacity-60"
            >
              End event
            </button>
          )}
          {activeAttempt && event.isHost && (
            <button
              onClick={handleRedeploy}
              disabled={busy}
              className="rounded-xl border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-muted hover:text-ink disabled:opacity-60"
            >
              Redeploy main
            </button>
          )}
          {activeAttempt && event.isHost && event.status === "ended" && (
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
          This event has no unbound tasks attached.
        </div>
      ) : (
        <>
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

          {event.status === "draft" ? (
            <div className="rounded-xl bg-surface p-6 text-muted">
              Start the event so participants can create solo or team attempts.
            </div>
          ) : (
            <>
              {activeTask && (
                <section className="rounded-xl bg-surface p-4">
                  <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <p className="text-xs font-semibold uppercase text-dim">
                        Attempts for {activeTask.title}
                      </p>
                      <p className="text-sm text-muted">
                        {event.isHost
                          ? "Every solo attempt and team gets its own repo and deployment."
                          : "Your solo or team workspace. Changes only deploy after you commit."}
                      </p>
                    </div>
                    {activeTask.attempts.length === 0 && event.status === "live" && (
                      <span className="text-xs text-dim">
                        No attempts yet — create one below.
                      </span>
                    )}
                  </div>

                  <div className="flex flex-wrap gap-2">
                    {activeTask.attempts.map((attempt) => (
                      <button
                        key={attempt.id}
                        onClick={() => setActiveAttemptId(attempt.id)}
                        className={`flex min-w-52 flex-col items-start gap-1 rounded-xl border px-3 py-2 text-left transition ${
                          attempt.id === activeAttemptId
                            ? "border-accent bg-accent/10"
                            : "border-line-hover bg-panel hover:bg-raised/60"
                        }`}
                      >
                        <span className="flex items-center gap-2 text-sm font-semibold text-ink">
                          {attempt.name}
                          {attempt.dirty && (
                            <span className="h-2 w-2 rounded-full bg-warning" title="Uncommitted changes" />
                          )}
                        </span>
                        <span className="text-xs text-muted">
                          {attempt.kind === "team" ? "team" : "solo"} ·{" "}
                          {attempt.owner?.username ?? "deleted user"}
                          {attempt.members.length > 1
                            ? ` +${attempt.members.length - 1}`
                            : ""}
                        </span>
                        <span className="flex items-center gap-2 text-xs">
                          <span
                            className={`rounded-full px-2 py-0.5 uppercase ${deploymentStyles[attempt.deployment.status] ?? "bg-raised text-dim"}`}
                          >
                            {attempt.deployment.status}
                          </span>
                          {attempt.deployment.commitSha && (
                            <span className="font-mono text-dim">
                              {attempt.deployment.commitSha.slice(0, 8)}
                            </span>
                          )}
                          {attempt.deployment.status === "live" && (
                            <a
                              href={attempt.deployment.url}
                              target="_blank"
                              rel="noreferrer"
                              onClick={(event) => event.stopPropagation()}
                              className="font-semibold text-accent-light hover:text-accent-lighter"
                            >
                              open
                            </a>
                          )}
                        </span>
                      </button>
                    ))}
                  </div>

                  {event.status === "live" && (
                    <div className="mt-4 flex flex-wrap items-end gap-3 border-t border-line pt-4">
                      <form onSubmit={handleCreateAttempt} className="flex flex-wrap items-end gap-2">
                        <div>
                          <label className="mb-1 block text-xs font-semibold uppercase text-dim">
                            New attempt
                          </label>
                          <div className="flex gap-2">
                            <select
                              value={attemptKind}
                              onChange={(event) => setAttemptKind(event.target.value as AttemptKind)}
                              className={`${inputClass} w-28`}
                            >
                              <option value="solo">Solo</option>
                              <option value="team">Team</option>
                            </select>
                            <input
                              value={attemptName}
                              onChange={(event) => setAttemptName(event.target.value)}
                              placeholder={attemptKind === "team" ? "team name" : "attempt name"}
                              className={`${inputClass} w-44`}
                            />
                            {event.isHost && (
                              <input
                                value={ownerUsername}
                                onChange={(event) => setOwnerUsername(event.target.value)}
                                placeholder="owner username (optional)"
                                className={`${inputClass} w-52`}
                              />
                            )}
                          </div>
                        </div>
                        <button
                          type="submit"
                          disabled={busy}
                          className="rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-shell hover:bg-accent-light disabled:opacity-60"
                        >
                          Create attempt
                        </button>
                      </form>

                      <form onSubmit={handleJoinTeam} className="flex items-end gap-2">
                        <div>
                          <label className="mb-1 block text-xs font-semibold uppercase text-dim">
                            Join a team
                          </label>
                          <input
                            value={teamCode}
                            onChange={(event) => setTeamCode(event.target.value)}
                            placeholder="invite code"
                            className={`${inputClass} w-36 font-mono uppercase`}
                          />
                        </div>
                        <button
                          type="submit"
                          disabled={busy}
                          className="rounded-xl border border-line-hover bg-panel px-3 py-2 text-sm font-semibold text-muted hover:text-ink disabled:opacity-60"
                        >
                          Join
                        </button>
                      </form>
                    </div>
                  )}
                </section>
              )}

              {activeAttempt ? (
                <div className="grid gap-4 xl:grid-cols-[220px_minmax(0,1fr)_minmax(0,1fr)]">
                  <aside className="flex min-h-64 flex-col rounded-xl bg-surface p-3">
                    <div className="mb-2 flex items-center justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold text-ink">
                          {activeAttempt.name}
                        </p>
                        <p className="truncate text-xs text-dim">
                          {activeAttempt.kind === "team" ? "team" : "solo"} workspace
                        </p>
                      </div>
                      <button
                        onClick={handleCreateFile}
                        disabled={readOnly}
                        className="rounded-lg bg-panel px-2 py-1 text-xs font-semibold text-muted hover:text-ink disabled:opacity-40"
                      >
                        + file
                      </button>
                    </div>
                    <ul className="flex flex-col overflow-auto">
                      {visiblePaths.map((path) => (
                        <li key={path} className="group flex items-center gap-1">
                          <button
                            onClick={() => openFile(path)}
                            className={`flex-1 truncate rounded-lg px-2 py-1 text-left font-mono text-xs transition hover:bg-raised/60 ${
                              activePath === path ? "text-accent-light" : "text-muted"
                            } ${binaryPaths.has(path) ? "opacity-60" : ""}`}
                          >
                            {path}
                            {docsRef.current.get(path)?.dirty ? " *" : ""}
                          </button>
                          <button
                            onClick={() => handleDeleteFile(path)}
                            disabled={readOnly}
                            className="hidden rounded px-1 text-xs text-dim hover:text-danger group-hover:block disabled:opacity-40"
                            title="Remove file"
                          >
                            ×
                          </button>
                        </li>
                      ))}
                      {visiblePaths.length === 0 && (
                        <li className="px-2 py-1 text-xs text-dim">No files</li>
                      )}
                    </ul>
                    <div className="mt-auto border-t border-line pt-3">
                      <p className="text-xs font-semibold uppercase text-dim">In the room</p>
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
                    </div>
                  </aside>

                  <section className="flex min-h-[60vh] flex-col overflow-hidden rounded-xl bg-surface">
                    <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line px-3 py-2">
                      <span className="truncate font-mono text-xs text-muted">
                        {activePath || "Select a file to edit"}
                      </span>
                      <span className="text-xs text-dim">
                        {readOnly ? "read-only" : dirty ? "uncommitted changes" : "clean"}
                      </span>
                    </div>
                    <Editor
                      height="60vh"
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
                    {activeAttempt.canEdit && (
                      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line px-3 py-2">
                        <p className="text-xs text-dim">
                          {dirty
                            ? "Edits stay in the live editor until you commit."
                            : "Workspace matches git main."}
                        </p>
                        <button
                          onClick={handleCommit}
                          disabled={busy || !dirty}
                          className="rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-shell hover:bg-accent-light disabled:opacity-50"
                        >
                          {busy ? "Committing…" : "Commit & deploy"}
                        </button>
                      </div>
                    )}
                  </section>

                  <section className="flex min-h-[60vh] flex-col rounded-xl bg-surface p-4">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div>
                        <p className="text-xs font-semibold uppercase text-dim">
                          Deployment (from git main)
                        </p>
                        <p className="text-sm font-semibold text-ink">
                          {activeAttempt.deployment.status === "live"
                            ? `main @ ${activeAttempt.deployment.commitSha.slice(0, 8)}`
                            : activeAttempt.deployment.status}
                        </p>
                      </div>
                      {activeAttempt.deployment.url && (
                        <a
                          href={activeAttempt.deployment.url}
                          target="_blank"
                          rel="noreferrer"
                          className="text-xs font-semibold text-accent-light hover:text-accent-lighter"
                        >
                          Open
                        </a>
                      )}
                    </div>
                    {dirty && (
                      <p className="mt-2 rounded-lg bg-warning/10 px-2 py-1 text-xs text-warning">
                        The editor has uncommitted changes. They are not part of
                        the deployment.
                      </p>
                    )}
                    {activeAttempt.deployment.status === "failed" && (
                      <p className="mt-2 whitespace-pre-wrap rounded-lg bg-danger/10 p-2 text-xs text-danger">
                        {activeAttempt.deployment.error}
                      </p>
                    )}
                    <div className="mt-3 flex-1 overflow-hidden rounded-lg bg-panel">
                      {activeAttempt.deployment.status === "live" ? (
                        <iframe
                          key={`${activeAttempt.deployment.commitSha}-${activeAttempt.deployment.deployedAt ?? ""}`}
                          src={activeAttempt.deployment.url}
                          title="Deployment preview"
                          sandbox="allow-scripts allow-forms allow-popups allow-modals"
                          className="h-[54vh] w-full bg-white"
                        />
                      ) : (
                        <div className="flex h-full min-h-40 items-center justify-center px-4 text-center text-sm text-muted">
                          Nothing deployed yet. Commit the workspace to main to
                          deploy it.
                        </div>
                      )}
                    </div>
                    {repoUrl && (
                      <button
                        onClick={() => copy(repoUrl)}
                        className="mt-2 truncate text-left font-mono text-xs text-dim hover:text-ink"
                        title="Copy clone URL"
                      >
                        {repoUrl}
                      </button>
                    )}
                  </section>
                </div>
              ) : (
                <div className="rounded-xl bg-surface p-6 text-muted">
                  {event.status === "live"
                    ? "Create a solo or team attempt to start building."
                    : "No attempt is visible for this task."}
                </div>
              )}

              {activeAttempt && (
                <section className="rounded-xl bg-surface p-5">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <h2 className="text-lg font-semibold text-ink">
                        {activeAttempt.name} members
                      </h2>
                      <p className="text-xs text-muted">
                        {activeAttempt.kind === "team"
                          ? "Team members share this workspace."
                          : "Solo attempt."}
                      </p>
                    </div>
                    {activeAttempt.joinCode && (
                      <button
                        onClick={() => copy(activeAttempt.joinCode!)}
                        className="rounded-full bg-panel px-3 py-1 font-mono text-xs text-muted hover:text-ink"
                        title="Copy team invite code"
                      >
                        invite: {activeAttempt.joinCode}
                      </button>
                    )}
                  </div>
                  <ul className="mt-3 flex flex-wrap gap-2">
                    {activeAttempt.members.map((member) => (
                      <li
                        key={member.userId}
                        className="flex items-center gap-2 rounded-full bg-panel px-3 py-1 text-sm text-muted"
                      >
                        <span className={member.role === "owner" ? "text-accent-light" : ""}>
                          {member.username}
                        </span>
                        <span className="text-xs uppercase text-dim">{member.role}</span>
                        {canManageAttempt && member.role !== "owner" && (
                          <button
                            onClick={() => handleRemoveAttemptMember(member)}
                            className="text-dim hover:text-danger"
                            title="Remove member"
                          >
                            ×
                          </button>
                        )}
                      </li>
                    ))}
                  </ul>
                  {canManageAttempt && activeAttempt.kind === "team" && (
                    <form onSubmit={handleAddAttemptMember} className="mt-3 flex items-center gap-2">
                      <input
                        value={memberName}
                        onChange={(event) => setMemberName(event.target.value)}
                        placeholder="username"
                        className={`${inputClass} w-48`}
                      />
                      <button
                        type="submit"
                        className="rounded-xl border border-line-hover bg-panel px-3 py-2 text-sm font-semibold text-muted hover:text-ink"
                      >
                        Add to team
                      </button>
                    </form>
                  )}
                </section>
              )}
            </>
          )}

          <section className="rounded-xl bg-surface p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-lg font-semibold text-ink">Event members</h2>
              {event.isHost && (
                <form onSubmit={handleAddEventMember} className="flex items-center gap-2">
                  <input
                    value={eventMemberName}
                    onChange={(input) => setEventMemberName(input.target.value)}
                    placeholder="username"
                    className={`${inputClass} w-40`}
                  />
                  <button
                    type="submit"
                    className="rounded-xl border border-line-hover bg-panel px-3 py-2 text-sm font-semibold text-muted hover:text-ink"
                  >
                    Add
                  </button>
                </form>
              )}
            </div>
            <ul className="mt-3 flex flex-wrap gap-2">
              {event.members.map((member) => (
                <li
                  key={member.userId}
                  className="flex items-center gap-2 rounded-full bg-panel px-3 py-1 text-sm text-muted"
                >
                  <span className={member.role === "host" ? "text-accent-light" : ""}>
                    {member.username}
                  </span>
                  <span className="text-xs uppercase text-dim">{member.role}</span>
                  {event.isHost && member.role !== "host" && (
                    <button
                      onClick={() => handleRemoveEventMember(member)}
                      className="text-dim hover:text-danger"
                      title="Remove member"
                    >
                      ×
                    </button>
                  )}
                </li>
              ))}
            </ul>
          </section>
        </>
      )}
    </main>
  );
}

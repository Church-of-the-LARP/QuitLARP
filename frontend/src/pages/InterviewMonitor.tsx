import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import usePageTitle from "../scripts/usePageTitle.ts";
import {
  endEvent,
  getEvent,
  getEventResults,
  getWorkspace,
  patchEvent,
  publishDeployment,
  deployAttempt,
  startEvent,
  type CollabDoc,
  type EventDetail,
  type EventTaskResults,
  type Workspace,
} from "../scripts/eventsApi.ts";

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

// InterviewMonitor is the HR-only monitoring page. It never exposes the
// participant editor or attempt controls: hosts only review participants,
// check runs, solutions and deployments.
export default function InterviewMonitor() {
  const params = useParams();
  const eventId = Number(params.id);
  usePageTitle("Interview monitor");

  const [event, setEvent] = useState<EventDetail | null>(null);
  const [results, setResults] = useState<EventTaskResults[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [preview, setPreview] = useState<{
    name: string;
    docs: CollabDoc[];
    files: Workspace["files"];
  } | null>(null);

  const load = async () => {
    try {
      const [loadedEvent, loadedResults] = await Promise.all([
        getEvent(eventId),
        getEventResults(eventId),
      ]);
      setEvent(loadedEvent);
      setResults(loadedResults.tasks ?? []);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load the interview");
    }
  };

  useEffect(() => {
    if (Number.isInteger(eventId) && eventId > 0) void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [eventId]);

  const handleStart = async () => {
    setBusy(true);
    try {
      await startEvent(eventId);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not start the interview");
    } finally {
      setBusy(false);
    }
  };

  const handleEnd = async () => {
    if (!window.confirm("End this interview? Attempts become read-only.")) return;
    setBusy(true);
    try {
      await endEvent(eventId);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not end the interview");
    } finally {
      setBusy(false);
    }
  };

  const toggleInvites = async () => {
    if (!event) return;
    setBusy(true);
    try {
      await patchEvent(eventId, { allowInvites: !event.allowInvites });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update invites");
    } finally {
      setBusy(false);
    }
  };

  const redeployAttempt = async (attemptId: number) => {
    setBusy(true);
    try {
      await deployAttempt(eventId, attemptId);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not redeploy");
    } finally {
      setBusy(false);
    }
  };

  const setPublished = async (attemptId: number, published: boolean) => {
    setBusy(true);
    try {
      await publishDeployment(eventId, attemptId, published);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update deployment");
    } finally {
      setBusy(false);
    }
  };

  const openAttemptCode = async (attemptId: number, name: string) => {
    try {
      const workspace = await getWorkspace(eventId, attemptId);
      setPreview({ name, docs: workspace.docs, files: workspace.files });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load the workspace");
    }
  };

  if (error && !event) {
    return (
      <main className="mx-auto w-[90vw] max-w-6xl py-8">
        <div
          role="alert"
          className="rounded-lg border border-danger/40 bg-danger/10 px-4 py-3 text-sm text-danger-light"
        >
          {error}
        </div>
        <Link
          to="/interviews"
          className="mt-4 inline-block text-sm font-semibold text-accent-light"
        >
          Back to interviews
        </Link>
      </main>
    );
  }

  if (!event) {
    return (
      <main className="mx-auto w-[90vw] max-w-6xl py-8 text-muted">
        Loading…
      </main>
    );
  }

  const attemptCount = results.reduce(
    (total, task) => total + task.attempts.length,
    0,
  );
  const participantCount = event.members.filter(
    (member) => member.role !== "host",
  ).length;

  return (
    <main className="mx-auto w-[90vw] max-w-6xl py-8 space-y-6">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <Link
            to="/interviews"
            className="inline-flex items-center gap-1 text-xs font-semibold uppercase text-accent-light"
          >
            <ArrowLeft aria-hidden="true" className="h-3.5 w-3.5" />
            Interviews
          </Link>
          <div className="mt-1 flex flex-wrap items-center gap-3">
            <h1 className="text-3xl font-bold text-ink">{event.title}</h1>
            <span
              className={`rounded-full px-2 py-0.5 text-xs font-semibold uppercase ${statusStyles[event.status] ?? "bg-raised text-muted"}`}
            >
              {event.status}
            </span>
            {event.joinCode && (
              <span className="rounded-full bg-panel px-3 py-1 font-mono text-xs text-muted">
                code: {event.joinCode}
              </span>
            )}
          </div>
          {event.description && (
            <p className="mt-2 max-w-2xl text-sm text-muted">{event.description}</p>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {event.isHost && event.status === "draft" && (
            <button
              onClick={() => void handleStart()}
              disabled={busy}
              className="rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-shell hover:bg-accent-light disabled:opacity-60"
            >
              Start interview
            </button>
          )}
          {event.isHost && event.status === "live" && (
            <button
              onClick={() => void handleEnd()}
              disabled={busy}
              className="rounded-xl border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-muted hover:text-ink disabled:opacity-60"
            >
              End interview
            </button>
          )}
          {event.isHost && (
            <button
              onClick={() => void toggleInvites()}
              disabled={busy}
              className={`rounded-xl border px-4 py-2 text-sm font-semibold disabled:opacity-60 ${
                event.allowInvites
                  ? "border-success/40 bg-success/10 text-success"
                  : "border-line-hover bg-panel text-muted hover:text-ink"
              }`}
            >
              Invites: {event.allowInvites ? "on" : "off"}
            </button>
          )}
          <Link
            to={`/interviews/${eventId}/room`}
            className="rounded-xl border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-accent-light hover:text-accent-lighter"
          >
            Enter room
          </Link>
          <button
            onClick={() => void load()}
            disabled={busy}
            className="rounded-lg bg-surface px-4 py-2 text-sm hover:bg-raised disabled:opacity-60"
          >
            Refresh
          </button>
        </div>
      </header>

      {error && (
        <div
          role="alert"
          className="rounded-lg border border-danger/40 bg-danger/10 px-4 py-3 text-sm text-danger-light"
        >
          {error}
        </div>
      )}

      <section className="grid gap-4 sm:grid-cols-3">
        <div className="rounded-xl bg-surface p-5">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted">
            Tasks
          </p>
          <p className="my-3 text-3xl font-semibold text-ink">
            {event.tasks.length}
          </p>
        </div>
        <div className="rounded-xl bg-surface p-5">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted">
            Participants
          </p>
          <p className="my-3 text-3xl font-semibold text-ink">
            {participantCount}
          </p>
        </div>
        <div className="rounded-xl bg-surface p-5">
          <p className="text-xs font-semibold uppercase tracking-wide text-muted">
            Workspaces
          </p>
          <p className="my-3 text-3xl font-semibold text-ink">
            {attemptCount}
          </p>
        </div>
      </section>

      {results.map((task) => (
        <section key={task.id} className="rounded-xl bg-surface p-5">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2 className="text-lg font-semibold text-ink">{task.title}</h2>
            <span className="rounded-full bg-raised px-2 py-0.5 text-xs uppercase text-dim">
              {task.kind}
            </span>
          </div>

          {task.kind === "unbound" ? (
            task.attempts.length === 0 ? (
              <p className="mt-3 rounded-lg bg-panel px-4 py-8 text-center text-sm text-muted">
                No workspaces yet.
              </p>
            ) : (
              <div className="mt-4 overflow-x-auto rounded-xl">
                <table className="w-full min-w-[720px] text-left text-sm">
                  <thead className="bg-panel text-xs uppercase text-muted">
                    <tr>
                      <th className="px-4 py-3 font-medium">Workspace</th>
                      <th className="px-4 py-3 font-medium">Members</th>
                      <th className="px-4 py-3 font-medium">Deployment</th>
                      <th className="px-4 py-3 font-medium">Commit</th>
                      <th className="px-4 py-3 font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {task.attempts.map((attempt) => (
                      <tr key={attempt.id}>
                        <td className="px-4 py-3 text-ink">{attempt.name}</td>
                        <td className="px-4 py-3 text-muted">
                          {attempt.members.map((member) => member.username).join(", ")}
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`rounded-full px-2 py-0.5 text-xs uppercase ${deploymentStyles[attempt.deployment.status] ?? "bg-raised text-dim"}`}
                          >
                            {attempt.deployment.status}
                          </span>
                        </td>
                        <td className="px-4 py-3 font-mono text-xs text-muted">
                          {attempt.deployment.commitSha.slice(0, 8) || "—"}
                        </td>
                        <td className="px-4 py-3">
                          <span className="flex flex-wrap gap-3 text-xs font-semibold">
                            <button
                              onClick={() => void openAttemptCode(attempt.id, attempt.name)}
                              className="text-accent-light hover:text-accent-lighter"
                            >
                              code
                            </button>
                            {attempt.deployment.url && (
                              <a
                                href={attempt.deployment.url}
                                target="_blank"
                                rel="noreferrer"
                                className="text-accent-light hover:text-accent-lighter"
                              >
                                deployment
                              </a>
                            )}
                            {event.status === "live" && (
                              <button
                                onClick={() => void redeployAttempt(attempt.id)}
                                disabled={busy}
                                className="text-accent-light hover:text-accent-lighter disabled:opacity-50"
                              >
                                redeploy
                              </button>
                            )}
                            {event.status === "ended" && (
                              <button
                                onClick={() =>
                                  void setPublished(
                                    attempt.id,
                                    !attempt.deployment.published,
                                  )
                                }
                                disabled={busy}
                                className="text-accent-light hover:text-accent-lighter disabled:opacity-50"
                              >
                                {attempt.deployment.published
                                  ? "unpublish"
                                  : "publish"}
                              </button>
                            )}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )
          ) : task.participants.length === 0 ? (
            <p className="mt-3 text-sm text-muted">No candidates assigned.</p>
          ) : (
            <div className="mt-3 flex flex-col gap-3">
              {task.participants.map((participant) => {
                const run = participant.testRun;
                const passed = run
                  ? run.results.filter((check) => check.passed).length
                  : 0;
                return (
                  <div
                    key={participant.userId}
                    className="rounded-xl bg-panel p-3"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
                      <span className="font-semibold text-ink">
                        {participant.username}
                      </span>
                      <span className="flex flex-wrap items-center gap-3 text-xs">
                        {run ? (
                          <span
                            className={
                              run.allPassed ? "text-success" : "text-warning"
                            }
                          >
                            {passed}/{run.results.length} checks
                          </span>
                        ) : (
                          <span className="text-dim">no checks yet</span>
                        )}
                        {participant.solution ? (
                          <span className="font-mono text-dim">
                            {participant.solution.commitSha.slice(0, 8)}
                          </span>
                        ) : (
                          <span className="text-dim">no solution</span>
                        )}
                      </span>
                    </div>
                    {participant.solution &&
                      participant.solution.files.length > 0 && (
                        <details className="mt-2">
                          <summary className="cursor-pointer text-xs font-semibold text-accent-light">
                            view code
                          </summary>
                          {participant.solution.files.map((file) => (
                            <div key={file.path} className="mt-2">
                              <p className="font-mono text-[11px] text-dim">
                                {file.path}
                              </p>
                              <pre className="max-h-72 overflow-auto whitespace-pre-wrap rounded bg-surface p-2 font-mono text-xs text-muted">
                                {file.content}
                              </pre>
                            </div>
                          ))}
                        </details>
                      )}
                  </div>
                );
              })}
            </div>
          )}
        </section>
      ))}

      {preview && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-shell/70 p-4">
          <div className="flex max-h-[85vh] w-full max-w-4xl flex-col rounded-xl border border-line-hover bg-panel">
            <div className="flex items-center justify-between border-b border-line px-4 py-3">
              <p className="font-semibold text-ink">{preview.name}</p>
              <button
                onClick={() => setPreview(null)}
                className="rounded px-2 py-1 text-sm text-muted hover:text-ink"
              >
                Close
              </button>
            </div>
            <div className="overflow-auto p-4">
              {preview.docs.map((doc) => (
                <div key={doc.path} className="mb-4">
                  <p className="font-mono text-xs text-dim">{doc.path}</p>
                  <pre className="mt-1 max-h-72 overflow-auto whitespace-pre-wrap rounded bg-surface p-2 font-mono text-xs text-muted">
                    {doc.content}
                  </pre>
                </div>
              ))}
              {preview.docs.length === 0 && (
                <p className="text-sm text-muted">No editable text files.</p>
              )}
              {preview.files.length > 0 && (
                <p className="mt-3 text-xs text-dim">
                  Other files: {preview.files.map((file) => file.path).join(", ")}
                </p>
              )}
            </div>
          </div>
        </div>
      )}
    </main>
  );
}

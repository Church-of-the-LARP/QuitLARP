import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import UsernameInput from "../components/UsernameInput.tsx";
import usePageTitle from "../scripts/usePageTitle.ts";
import useAuth from "../scripts/useAuth.tsx";
import {
  createEvent,
  listEvents,
  listHostableTasks,
  type EventSummary,
  type UnboundTask,
} from "../scripts/eventsApi.ts";

const inputClass =
  "w-full rounded-xl border border-line-hover bg-panel px-3 py-2 text-ink outline-none placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30";

const statusStyles: Record<string, string> = {
  draft: "bg-raised text-muted",
  live: "bg-success/20 text-success",
  ended: "bg-accent-secondary/20 text-accent-secondary",
};

// EventsHost is the HR-only surface: create monitored interview sessions and
// open them to review workspaces, deployments and check runs.
export default function EventsHost() {
  usePageTitle("Host interviews");
  const navigate = useNavigate();
  const { user } = useAuth();
  const [events, setEvents] = useState<EventSummary[]>([]);
  const [tasks, setTasks] = useState<UnboundTask[]>([]);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [selected, setSelected] = useState<number[]>(() => {
    const raw = new URLSearchParams(window.location.search).get("task");
    const taskID = Number(raw);
    return Number.isInteger(taskID) && taskID > 0 ? [taskID] : [];
  });
  const [memberUsernames, setMemberUsernames] = useState("");
  const [allowInvites, setAllowInvites] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    listEvents()
      .then((list) =>
        setEvents(
          list.filter(
            (event) => event.monitored && event.host?.id === user?.id,
          ),
        ),
      )
      .catch(() => setEvents([]));
    listHostableTasks()
      .then(setTasks)
      .catch(() => setTasks([]));
  }, [user?.id]);

  const toggleTask = (id: number) => {
    setSelected((current) =>
      current.includes(id)
        ? current.filter((entry) => entry !== id)
        : [...current, id],
    );
  };

  const handleCreate = async (submit: React.FormEvent<HTMLFormElement>) => {
    submit.preventDefault();
    setError("");
    if (!title.trim()) {
      setError("Give the interview a title.");
      return;
    }
    setBusy(true);
    try {
      const created = await createEvent({
        title: title.trim(),
        description: description.trim(),
        taskIds: selected,
        monitored: true,
        allowInvites,
        memberUsernames: memberUsernames
          .split(/[,\s]+/)
          .map((name) => name.trim())
          .filter(Boolean),
      });
      navigate(`/interviews/${created.id}`);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Could not create the interview",
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="mx-auto flex w-[calc(100%-2rem)] flex-col gap-6 py-6 lg:w-[75vw]">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
            HR interviews
          </p>
          <h1 className="text-3xl font-bold text-ink">Host interviews</h1>
          <p className="mt-2 max-w-2xl text-sm text-muted">
            Create a monitored session, assign people, then review every
            workspace, deployment, solution and check run.
          </p>
        </div>
      </header>

      {error && (
        <div className="rounded-xl bg-danger/15 px-4 py-3 text-sm text-danger">
          {error}
        </div>
      )}

      {events.length === 0 ? (
        <p className="text-sm text-muted">
          No monitored interviews yet. Create one below.
        </p>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {events.map((event) => (
            <Link
              key={event.id}
              to={`/interviews/${event.id}`}
              className="flex flex-col gap-3 rounded-xl bg-surface p-5 transition hover:bg-raised/60"
            >
              <div className="flex items-start justify-between gap-3">
                <h2 className="text-lg font-semibold text-ink">{event.title}</h2>
                <span
                  className={`rounded-full px-2 py-0.5 text-xs font-semibold uppercase ${statusStyles[event.status] ?? "bg-raised text-muted"}`}
                >
                  {event.status}
                </span>
              </div>
              {event.description && (
                <p className="line-clamp-2 text-sm text-muted">
                  {event.description}
                </p>
              )}
              <p className="mt-auto text-xs text-dim">
                {event.taskCount} task{event.taskCount === 1 ? "" : "s"} · open
                to review results
              </p>
            </Link>
          ))}
        </div>
      )}

      <section className="rounded-xl border border-line-hover bg-surface p-6">
        <h2 className="text-xl font-semibold text-ink">Host an interview</h2>
        <p className="mt-1 text-sm text-muted">
          Pick the assessments to include and assign candidates. Their latest
          solutions and check runs appear in the interview results.
        </p>
        <form onSubmit={handleCreate} className="mt-5 flex flex-col gap-4">
          <div>
            <label className="mb-1 block text-sm font-semibold text-muted" htmlFor="interview-title">
              Title
            </label>
            <input
              id="interview-title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="e.g. Backend interview — Two Sum"
              className={inputClass}
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-semibold text-muted" htmlFor="interview-description">
              Description
            </label>
            <textarea
              id="interview-description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              rows={3}
              className={inputClass}
            />
          </div>
          <div>
            <p className="mb-2 text-sm font-semibold text-muted">Tasks</p>
            {tasks.length === 0 ? (
              <p className="text-sm text-muted">No assessments available yet.</p>
            ) : (
              <div className="grid gap-2 sm:grid-cols-2">
                {tasks.map((task) => (
                  <label
                    key={task.id}
                    className="flex cursor-pointer items-start gap-3 rounded-xl bg-panel px-3 py-2"
                  >
                    <input
                      type="checkbox"
                      checked={selected.includes(task.id)}
                      onChange={() => toggleTask(task.id)}
                      className="mt-1 accent-accent"
                    />
                    <span className="min-w-0">
                      <span className="flex items-center gap-2 text-sm font-semibold text-ink">
                        <span className="truncate">{task.title}</span>
                        <span className="shrink-0 rounded-full bg-raised px-2 py-0.5 text-[10px] uppercase text-dim">
                          {task.kind}
                        </span>
                      </span>
                      <span className="mt-0.5 line-clamp-3 text-xs text-muted">
                        {task.description}
                      </span>
                    </span>
                  </label>
                ))}
              </div>
            )}
          </div>
          <div>
            <label className="mb-1 block text-sm font-semibold text-muted" htmlFor="interview-members">
              Assign people
            </label>
            <UsernameInput
              id="interview-members"
              value={memberUsernames}
              onChange={setMemberUsernames}
              placeholder="usernames, separated by commas"
              className={inputClass}
            />
            <p className="mt-1 text-xs text-dim">
              Assigned users see the interview under Upcoming tests until its end
              time passes.
            </p>
          </div>
          <label className="flex items-center gap-2 text-sm text-muted">
            <input
              type="checkbox"
              checked={allowInvites}
              onChange={(event) => setAllowInvites(event.target.checked)}
              className="accent-accent"
            />
            Allow candidates to invite collaborators into their attempt
          </label>
          <button
            type="submit"
            disabled={busy}
            className="self-start rounded-xl bg-accent px-5 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light disabled:opacity-60"
          >
            {busy ? "Creating…" : "Create interview"}
          </button>
        </form>
      </section>
    </main>
  );
}

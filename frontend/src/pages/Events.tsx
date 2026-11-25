import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import usePageTitle from "../scripts/usePageTitle.ts";
import {
  createEvent,
  joinEvent,
  listEvents,
  listUnboundTasks,
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

export default function Events() {
  usePageTitle("Events");
  const navigate = useNavigate();
  const [events, setEvents] = useState<EventSummary[]>([]);
  const [tasks, setTasks] = useState<UnboundTask[]>([]);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [selected, setSelected] = useState<number[]>([]);
  const [joinCode, setJoinCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    listEvents()
      .then(setEvents)
      .catch(() => setEvents([]));
    listUnboundTasks()
      .then(setTasks)
      .catch(() => setTasks([]));
  }, []);

  const toggleTask = (id: number) => {
    setSelected((current) =>
      current.includes(id)
        ? current.filter((entry) => entry !== id)
        : [...current, id],
    );
  };

  const handleCreate = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    if (!title.trim()) {
      setError("Give the event a title.");
      return;
    }
    setBusy(true);
    try {
      const created = await createEvent({
        title: title.trim(),
        description: description.trim(),
        taskIds: selected,
      });
      navigate(`/events/${created.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not create the event");
    } finally {
      setBusy(false);
    }
  };

  const handleJoin = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    if (!joinCode.trim()) return;
    setBusy(true);
    try {
      const joined = await joinEvent(joinCode.trim());
      navigate(`/events/${joined.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not join the event");
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="mx-auto flex w-[calc(100%-2rem)] flex-col gap-6 py-6 lg:w-[75vw]">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
            Collaborative sessions
          </p>
          <h1 className="text-3xl font-bold text-ink">Events</h1>
          <p className="mt-2 max-w-2xl text-sm text-muted">
            Host an interview or competition on unbound web-app tasks. Members
            edit and push together in the browser; main deploys to a live
            preview the host can watch next to the code.
          </p>
        </div>
        <form onSubmit={handleJoin} className="flex items-center gap-2">
          <input
            value={joinCode}
            onChange={(event) => setJoinCode(event.target.value)}
            placeholder="Join code"
            aria-label="Join code"
            className={`${inputClass} w-40 font-mono uppercase`}
          />
          <button
            type="submit"
            disabled={busy}
            className="rounded-xl border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-muted transition hover:text-ink disabled:opacity-60"
          >
            Join
          </button>
        </form>
      </header>

      {error && (
        <div className="rounded-xl bg-danger/15 px-4 py-3 text-sm text-danger">
          {error}
        </div>
      )}

      {events.length === 0 ? (
        <p className="text-sm text-muted">
          No events yet. Create one below, or join with a code from a host.
        </p>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {events.map((event) => (
            <Link
              key={event.id}
              to={`/events/${event.id}`}
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
                Hosted by {event.host?.username ?? "deleted user"} ·{" "}
                {event.taskCount} task{event.taskCount === 1 ? "" : "s"}
              </p>
            </Link>
          ))}
        </div>
      )}

      <section className="rounded-xl border border-line-hover bg-surface p-6">
        <h2 className="text-xl font-semibold text-ink">Host an event</h2>
        <p className="mt-1 text-sm text-muted">
          Pick the unbound tasks members will build. Workspaces are created and
          deployed when you start the event.
        </p>
        <form onSubmit={handleCreate} className="mt-5 flex flex-col gap-4">
          <div>
            <label className="mb-1 block text-sm font-semibold text-muted" htmlFor="event-title">
              Title
            </label>
            <input
              id="event-title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="e.g. Frontend interview — build a kanban board"
              className={inputClass}
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-semibold text-muted" htmlFor="event-description">
              Description
            </label>
            <textarea
              id="event-description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              rows={3}
              className={inputClass}
            />
          </div>
          <div>
            <p className="mb-2 text-sm font-semibold text-muted">Tasks</p>
            {tasks.length === 0 ? (
              <p className="text-sm text-muted">
                No unbound tasks yet.{" "}
                <Link to="/create?kind=unbound" className="font-semibold text-accent-light hover:text-accent-lighter">
                  Create one
                </Link>{" "}
                by pushing an app with a <code className="font-mono">deploy/</code>{" "}
                directory.
              </p>
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
                    <span>
                      <span className="block text-sm font-semibold text-ink">
                        {task.title}
                      </span>
                      <span className="block line-clamp-2 text-xs text-muted">
                        {task.description}
                      </span>
                    </span>
                  </label>
                ))}
              </div>
            )}
          </div>
          <button
            type="submit"
            disabled={busy}
            className="self-start rounded-xl bg-accent px-5 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light disabled:opacity-60"
          >
            {busy ? "Creating…" : "Create event"}
          </button>
        </form>
      </section>
    </main>
  );
}

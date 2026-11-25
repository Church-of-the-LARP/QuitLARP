import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import usePageTitle from "../scripts/usePageTitle.ts";
import {
  createEvent,
  joinEvent,
  listHostableTasks,
  type UnboundTask,
} from "../scripts/eventsApi.ts";

const inputClass =
  "w-full rounded-xl border border-line-hover bg-panel px-3 py-2 text-ink outline-none placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30";

export default function Events() {
  usePageTitle("Collaborations");
  const navigate = useNavigate();
  const [tasks, setTasks] = useState<UnboundTask[]>([]);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [selected, setSelected] = useState<number[]>(() => {
    const raw = new URLSearchParams(window.location.search).get("task");
    const taskID = Number(raw);
    return Number.isInteger(taskID) && taskID > 0 ? [taskID] : [];
  });
  const [joinCode, setJoinCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    listHostableTasks()
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
      setError("Give the collaboration a title.");
      return;
    }
    setBusy(true);
    try {
      const created = await createEvent({
        title: title.trim(),
        description: description.trim(),
        taskIds: selected,
      });
      navigate(`/collaborations/${created.id}`);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Could not create the collaboration",
      );
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
      navigate(`/collaborations/${joined.id}`);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Could not join the collaboration",
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
            Collaborative sessions
          </p>
          <h1 className="text-3xl font-bold text-ink">Collaborations</h1>
          <p className="mt-2 max-w-2xl text-sm text-muted">
            Join with a code from your host, or create a collaboration on any
            assessment. Unbound web-app tasks get shared workspaces; coding
            tasks open as normal practice runs.
          </p>
        </div>
        <form onSubmit={handleJoin} className="flex items-center gap-2">
          <input
            value={joinCode}
            onChange={(event) => setJoinCode(event.target.value)}
            placeholder="Collaboration code"
            aria-label="Collaboration code"
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

      <p className="text-sm text-muted">
        Collaborations are private. Share the collaboration code with the people
        who should join.
      </p>

      <section className="rounded-xl border border-line-hover bg-surface p-6">
        <h2 className="text-xl font-semibold text-ink">
          Host a collaboration
        </h2>
        <p className="mt-1 text-sm text-muted">
          Pick the assessments to collaborate on. The collaboration is private:
          share its code with the people who should join.
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
              placeholder="e.g. Pair on a landing page"
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
                No assessments are available yet.
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
          <button
            type="submit"
            disabled={busy}
            className="self-start rounded-xl bg-accent px-5 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light disabled:opacity-60"
          >
            {busy ? "Creating…" : "Create collaboration"}
          </button>
        </form>
      </section>
    </main>
  );
}

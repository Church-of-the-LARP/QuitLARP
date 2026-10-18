import { useEffect, useRef, useState } from "react";
import Alert from "../components/Alert.tsx";
import useAuth from "../scripts/useAuth.tsx";
import type { ActionResult } from "../types/types.ts";
import usePageTitle from "../scripts/usePageTitle.ts";
import { useNotifications } from "../scripts/useNotifications.ts";
import {
  getAssessments,
  getScheduled,
  getNotifications,
  type Notification,
  type ScheduledAssessment,
  getTags,
  type AssessmentSummary,
  type Difficulty,
  type TagCount,
} from "../scripts/dashApi.ts";

function timeAgo(iso: string) {
  const mins = Math.floor((Date.now() - new Date(iso).getTime()) / 60000);
  if (mins < 1) return "Just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function VerifyEmailBanner({
  setCloseEmail,
}: {
  setCloseEmail: (input: boolean) => void;
}) {
  const { user, resendVerification } = useAuth();
  const [alertMsg, setAlertMsg] = useState<ActionResult>({});

  const handleResend = async () => {
    setAlertMsg(await resendVerification());
  };

  return (
    <div className="mb-6 rounded-md border border-warning bg-warning/80 px-4 py-3 text-sm text-shell">
      <p className="font-medium">Verify your email address</p>
      <p className="mt-0.5 text-shell/80">
        We emailed a verification link to{" "}
        <span className="font-semibold">{user?.email}</span>. Didn't get it?{" "}
        <button
          onClick={() => handleResend()}
          className="font-semibold underline underline-offset-2 hover:text-panel"
        >
          Resend the email
        </button>
        .
      </p>
      <button
        className="absolute top-1 right-3 p-2 hover:text-panel"
        onClick={() => setCloseEmail(true)}
      >
        X
      </button>
      {alertMsg.error && <Alert kind="error" text={alertMsg.error} />}
      {alertMsg.notice && <Alert kind="notice" text={alertMsg.notice} />}
    </div>
  );
}

function App() {
  usePageTitle("Dashboard");
  const emptyFilters: Filters = { q: "", author: "", difficulty: "", tag: "" };
  const { user } = useAuth();
  const [closeEmail, setCloseEmail] = useState(false);
  const [notifications, setNotifications] = useState<Notification[]>([]);

  useEffect(() => {
    getNotifications()
      .then(setNotifications)
      .catch(() => setNotifications([]));
  }, []);

  // the backend pushes {type: "notification", data: <Notification>}
  useNotifications((n) => {
    if (n.type !== "notification") return;
    setNotifications((prev) => [n.data as Notification, ...prev]);
  }, !!user);
  const [draft, setDraft] = useState<Filters>(emptyFilters);
  const firstLoad = useRef(true);
  const [publicTests, setPublicTests] = useState<AssessmentSummary[]>([]);
  const [upcomingTests, setUpcomingTests] = useState<ScheduledAssessment[]>([]);
  const [testCount, setTestCount] = useState(0);
  const [offset, setOffset] = useState(0);
  const [tags, setTags] = useState<TagCount[]>([]);
  const [loadError, setLoadError] = useState("");

  const PAGE_SIZE = 5;
  const difficulties: Difficulty[] = ["easy", "medium", "hard"];

  type Filters = {
    q: string;
    author: string;
    difficulty: Difficulty | "";
    tag: string;
  };

  const difficultyColor: Record<string, string> = {
    easy: "text-success",
    medium: "text-warning",
    hard: "text-danger",
  };

  useEffect(() => {
    getScheduled()
      .then(setUpcomingTests)
      .catch(() => setUpcomingTests([]));
  }, []);

  useEffect(() => {
    getTags()
      .then(setTags)
      .catch(() => setTags([]));
  }, []);

  useEffect(() => {
    const timer = setTimeout(
      () => {
        firstLoad.current = false;
        getAssessments({
          q: draft.q || undefined,
          author: draft.author.trim() || undefined,
          difficulty: draft.difficulty || undefined,
          tag: draft.tag.trim() || undefined,
          offset: offset,
          limit: PAGE_SIZE,
        })
          .then((res) => {
            setPublicTests(res.assessments);
            setTestCount(res.total);
            setLoadError("");
          })
          .catch((e: Error) => setLoadError(e.message));
      },
      firstLoad.current ? 0 : 300,
    );
    return () => clearTimeout(timer);
  }, [draft, offset]);

  const handleReset = () => setDraft(emptyFilters);

  if (!user) return null;

  return (
    <>
      <div className="min-w-screen flex flex-col justify-center items-center">
        <div className="max-w-3xl fixed top-12">
          {!user.emailVerified && !closeEmail ? (
            <VerifyEmailBanner setCloseEmail={setCloseEmail} />
          ) : undefined}
        </div>
      </div>

      <main className="mx-auto w-[75vw] flex flex-col gap-5 py-6">
        <section className="mb-6">
          <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
            Your progress
          </p>
          <h1 className="mb-6 text-3xl font-bold text-ink">User stats</h1>
          <div className="flex gap-5 w-full h-60">
            <div className="flex h-full w-full flex-col justify-center rounded-xl border border-line-hover bg-surface px-5 text-center">
              <p className="text-sm font-semibold uppercase text-muted">
                Completed
              </p>
              <p className="my-2 text-5xl font-bold text-ink">0</p>
              <p className="text-sm font-medium text-muted">tests</p>
            </div>
            <div className="flex h-full w-full flex-col justify-center rounded-xl border border-line-hover bg-surface px-5 text-center">
              <p className="text-sm font-semibold uppercase text-muted">
                In progress
              </p>
              <p className="my-2 text-5xl font-bold text-ink">0</p>
              <p className="text-sm font-medium text-muted">tests</p>
            </div>
            <div className="flex h-full w-full flex-col justify-center rounded-xl border border-line-hover bg-surface px-5 text-center">
              <p className="text-sm font-semibold uppercase text-muted">
                Average score
              </p>
              <p className="my-2 text-5xl font-bold text-ink">0%</p>
              <p className="text-sm font-medium text-muted">across tests</p>
            </div>
          </div>
        </section>

        <section className="my-6">
          <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
            Keep learning
          </p>
          <h1 className="mb-6 text-3xl font-bold text-ink">Upcoming tests</h1>
          <div className="flex flex-nowrap overflow-auto justify-start items-center gap-5">
            {upcomingTests.length > 0 ? (
              upcomingTests.map((test, idx) => (
                <div
                  className="flex flex-col w-96 shrink-0 rounded-xl border border-line-hover bg-surface p-5 md:flex-row"
                  key={idx}
                >
                  <div className="w-full">
                    <h2 className="text-lg font-semibold text-ink">
                      {test.title}
                    </h2>
                    <p className="mt-2 text-sm font-medium text-muted">
                      {test.scheduledAt}
                    </p>
                    <p className="mt-1 text-sm text-muted">
                      {test.timeLimitMinutes} minutes
                    </p>
                  </div>
                  <div className="w-full md:w-40 text-center content-center md:content-end md:text-end">
                    <button className="rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light">
                      Take test
                    </button>
                  </div>
                </div>
              ))
            ) : (
              <div className="w-full flex items-center justify-center h-20">
                <p>No upcoming tests</p>
              </div>
            )}
          </div>
        </section>

        <div className="flex gap-5">
          <section className="w-[70%]">
            <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
              Explore
            </p>
            <h1 className="mb-5 text-3xl font-bold text-ink">Public tests</h1>
            <form
              className="mb-5 grid grid-cols-1 md:grid-cols-2 lg:flex lg:flex-wrap items-end gap-3 w-full justify-around"
              onSubmit={(e) => e.preventDefault()}
            >
              <div className="flex flex-1 flex-col gap-1">
                <label className="text-sm font-semibold text-muted">
                  Search
                </label>
                <input
                  type="text"
                  value={draft.q}
                  onChange={(e) => setDraft({ ...draft, q: e.target.value })}
                  className="w-full rounded-lg border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30"
                  placeholder="Search for title"
                />
              </div>
              <div className="flex flex-1 flex-col gap-1">
                <label className="text-sm font-semibold text-muted">
                  Author
                </label>
                <input
                  type="text"
                  value={draft.author}
                  onChange={(e) =>
                    setDraft({ ...draft, author: e.target.value })
                  }
                  className="w-full rounded-lg border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30"
                  placeholder="Search for author"
                />
              </div>
              <div className="flex flex-1 flex-col gap-1">
                <label className="text-sm font-semibold text-muted">
                  Difficulty
                </label>
                <select
                  value={draft.difficulty}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      difficulty: e.target.value as Difficulty | "",
                    })
                  }
                  className="w-full rounded-lg border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30"
                >
                  <option value="">Any</option>
                  {difficulties.map((d) => (
                    <option key={d} value={d}>
                      {d}
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex flex-1 flex-col gap-1">
                <label className="text-sm font-semibold text-muted">Tag</label>
                <input
                  type="text"
                  list="tag-options"
                  value={draft.tag}
                  onChange={(e) => setDraft({ ...draft, tag: e.target.value })}
                  className="w-full rounded-lg border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30"
                  placeholder="Select or type a tag"
                />
                <datalist id="tag-options">
                  {tags.map((t) => (
                    <option key={t.id} value={t.name} />
                  ))}
                </datalist>
              </div>
              <button
                type="button"
                onClick={handleReset}
                className="md:col-span-2 rounded-lg border border-line-hover px-4 py-2 text-sm font-semibold text-muted transition hover:text-ink"
              >
                Reset
              </button>
            </form>

            {loadError && <Alert kind="error" text={loadError} />}
            {!loadError && publicTests.length === 0 && (
              <p className="text-sm text-muted">No tests found.</p>
            )}

            <div className="flex flex-col gap-5">
              {publicTests.map((test) => (
                <div
                  className="w-full rounded-xl border border-line-hover bg-surface p-5 flex flex-col md:flex-row"
                  key={test.id}
                >
                  <div className="w-full">
                    <h2 className="text-lg font-semibold text-ink">
                      {test.title}
                    </h2>
                    <p className="mt-2 text-sm text-muted">
                      By {test.author?.username ?? "deleted user"}
                    </p>
                    <p
                      className={`mt-1 text-xs font-semibold uppercase ${difficultyColor[test.difficulty] ?? "text-muted"}`}
                    >
                      {test.difficulty}
                    </p>
                  </div>
                  <div className="w-full md:w-30 text-center content-center md:content-end md:text-end">
                    <button className="rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light">
                      Take test
                    </button>
                  </div>
                </div>
              ))}
            </div>
            <div className="flex items-center justify-between gap-2 mt-5">
              <button
                className="rounded-lg border border-line-hover bg-surface px-4 py-2 text-sm font-semibold text-ink transition hover:bg-accent hover:text-shell disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-surface disabled:hover:text-ink"
                disabled={offset <= 0}
                onClick={() => setOffset((prev) => prev - 5)}
              >
                «
              </button>
              <button
                className="rounded-lg border border-line-hover bg-surface px-4 py-2 text-sm font-semibold text-ink transition hover:bg-accent hover:text-shell disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-surface disabled:hover:text-ink"
                disabled={offset + 5 >= testCount}
                onClick={() => setOffset((prev) => prev + 5)}
              >
                »
              </button>
            </div>
          </section>

          <aside className="w-[30%]">
            <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
              Updates
            </p>
            <h1 className="mb-6 text-3xl font-bold text-ink">Notifications</h1>
            <div className="flex flex-col gap-5">
              {notifications.length === 0 && (
                <div className="w-full flex items-center justify-center h-20">
                  <p>No notifications yet.</p>
                </div>
              )}
              {notifications.map((notification) => (
                <div
                  className="w-full rounded-xl border border-line-hover bg-surface p-5"
                  key={notification.id}
                >
                  <h2 className="text-base font-semibold text-ink">
                    {notification.title}
                  </h2>
                  <p className="mt-2 text-sm leading-5 text-muted">
                    {notification.detail}
                  </p>
                  <p className="mt-3 text-xs font-medium uppercase text-dim">
                    {timeAgo(notification.createdAt)}
                  </p>
                </div>
              ))}
            </div>
          </aside>
        </div>
      </main>
    </>
  );
}

export default App;

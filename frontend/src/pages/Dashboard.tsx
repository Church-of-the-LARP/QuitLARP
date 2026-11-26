import { useState, useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import { ChevronDown, X } from "lucide-react";
import Alert from "../components/Alert.tsx";
import useAuth from "../scripts/useAuth.tsx";
import type { ActionResult } from "../types/types.ts";
import usePageTitle from "../scripts/usePageTitle.ts";
import {
  getAssessments,
  getNotifications,
  getScheduled,
  type Notification,
  type ScheduledAssessment,
  getTags,
  type AssessmentSummary,
  type Difficulty,
  type TagCount,
} from "../scripts/dashApi.ts";
import { listUpcomingEvents, type EventSummary } from "../scripts/eventsApi.ts";

function timeAgo(value: string) {
  const seconds = Math.max(
    1,
    Math.floor((Date.now() - new Date(value).getTime()) / 1000),
  );
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  return new Date(value).toLocaleDateString();
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
    <div className="relative mb-6 rounded-md bg-warning/80 px-4 py-3 text-sm text-shell">
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
        aria-label="Dismiss"
        title="Dismiss"
        className="absolute top-1 right-3 rounded p-2 hover:text-panel"
        onClick={() => setCloseEmail(true)}
      >
        <X aria-hidden="true" className="h-4 w-4" />
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
  const [draft, setDraft] = useState<Filters>(emptyFilters);
  const firstLoad = useRef(true);
  const [publicTests, setPublicTests] = useState<AssessmentSummary[]>([]);
  const [upcomingTests, setUpcomingTests] = useState<ScheduledAssessment[]>([]);
  const [upcomingEvents, setUpcomingEvents] = useState<EventSummary[]>([]);
  const [testCount, setTestCount] = useState(0);
  const [offset, setOffset] = useState(0);
  const [tags, setTags] = useState<TagCount[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [notificationsLoading, setNotificationsLoading] = useState(true);
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
    listUpcomingEvents()
      .then(setUpcomingEvents)
      .catch(() => setUpcomingEvents([]));
  }, []);

  useEffect(() => {
    getTags()
      .then(setTags)
      .catch(() => setTags([]));
  }, []);

  useEffect(() => {
    getNotifications(20)
      .then(setNotifications)
      .catch(() => setNotifications([]))
      .finally(() => setNotificationsLoading(false));
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
        <div className="max-w-3xl fixed top-12 px-4 z-30">
          {!user.emailVerified && !closeEmail ? (
            <VerifyEmailBanner setCloseEmail={setCloseEmail} />
          ) : undefined}
        </div>
      </div>

      <main className="mx-auto w-[calc(100%-2rem)] sm:w-[75vw] flex flex-col gap-5 py-6">
        <section className="mb-6">
          <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
            <div>
              <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
                Your progress
              </p>
              <h1 className="text-3xl font-bold text-ink">User stats</h1>
            </div>
            <div className="flex items-center gap-2">
              <Link
                to="/collaborations"
                className="rounded-lg border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-muted transition hover:text-ink"
              >
                Collaboration
              </Link>
              {(user.role === "moderator" || user.role === "admin") && (
                <Link
                  to="/interviews"
                  className="rounded-lg border border-line-hover bg-panel px-4 py-2 text-sm font-semibold text-muted transition hover:text-ink"
                >
                  Host
                </Link>
              )}
              {(user.role === "moderator" || user.role === "admin") && (
                <Link
                  to="/create"
                  className="rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell"
                >
                  Create assessment
                </Link>
              )}
            </div>
          </div>
          <div className="flex flex-col sm:flex-row gap-5 w-full sm:h-60">
            <div className="flex h-full w-full flex-col justify-center rounded-xl bg-surface px-5 py-5 text-center">
              <p className="text-sm font-semibold uppercase text-muted">
                Completed
              </p>
              <p className="my-2 text-5xl font-bold text-ink">0</p>
              <p className="text-sm font-medium text-muted">tests</p>
            </div>
            <div className="flex h-full w-full flex-col justify-center rounded-xl bg-surface px-5 py-5 text-center">
              <p className="text-sm font-semibold uppercase text-muted">
                In progress
              </p>
              <p className="my-2 text-5xl font-bold text-ink">0</p>
              <p className="text-sm font-medium text-muted">tests</p>
            </div>
            <div className="flex h-full w-full flex-col justify-center rounded-xl bg-surface px-5 py-5 text-center">
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
          <h1 className="mb-6 text-3xl font-bold text-ink">
            Upcoming tests
          </h1>
          <div className="flex flex-nowrap overflow-auto justify-start items-center gap-5">
            {upcomingEvents.length > 0 || upcomingTests.length > 0 ? (
              <>
                {upcomingEvents.map((upcomingEvent) => (
                  <div
                    className="flex w-72 shrink-0 flex-col rounded-xl bg-surface p-5 sm:w-96 md:flex-row"
                    key={`event-${upcomingEvent.id}`}
                  >
                    <div className="w-full min-w-0">
                      <h2 className="truncate text-lg font-semibold text-ink">
                        {upcomingEvent.title}
                      </h2>
                      {upcomingEvent.description && (
                        <p className="mt-1 line-clamp-2 text-xs text-muted">
                          {upcomingEvent.description}
                        </p>
                      )}
                      <p className="mt-2 text-sm font-medium text-muted">
                        {upcomingEvent.endsAt
                          ? `Ends ${new Date(upcomingEvent.endsAt).toLocaleString()}`
                          : "No deadline"}
                      </p>
                      <p className="mt-1 text-sm text-muted">
                        {upcomingEvent.taskCount} task
                        {upcomingEvent.taskCount === 1 ? "" : "s"}
                      </p>
                    </div>
                    <div className="w-full content-center text-center md:w-32 md:content-end md:text-end">
                      <Link
                        to={
                          upcomingEvent.monitored
                            ? upcomingEvent.host?.id === user?.id
                              ? `/interviews/${upcomingEvent.id}`
                              : `/interviews/${upcomingEvent.id}/room`
                            : `/collaborations/${upcomingEvent.id}`
                        }
                        className="rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light"
                      >
                        {upcomingEvent.monitored &&
                        upcomingEvent.host?.id === user?.id
                          ? "Monitor"
                          : "Open"}
                      </Link>
                    </div>
                  </div>
                ))}
                {upcomingTests.map((test, idx) => (
                  <div
                    className="flex flex-col w-72 sm:w-96 shrink-0 rounded-xl bg-surface p-5 md:flex-row"
                    key={`scheduled-${test.id}-${idx}`}
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
                      <Link
                        to={`/assessments/${test.assessmentId}/preview`}
                        className="rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light"
                      >
                        Take test
                      </Link>
                    </div>
                  </div>
                ))}
              </>
            ) : (
              <div className="w-full flex items-center justify-center h-20">
                <p>No upcoming tests</p>
              </div>
            )}
          </div>
        </section>

        <div className="flex flex-col lg:flex-row gap-5">
          <section className="w-full lg:w-[70%]">
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
                  className="w-full rounded-lg bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:ring-2 focus:ring-accent/30"
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
                  className="w-full rounded-lg bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:ring-2 focus:ring-accent/30"
                  placeholder="Search for author"
                />
              </div>
              <div className="flex flex-1 flex-col gap-1">
                <label className="text-sm font-semibold text-muted">
                  Difficulty
                </label>
                <div className="relative">
                  <select
                    value={draft.difficulty}
                    onChange={(e) =>
                      setDraft({
                        ...draft,
                        difficulty: e.target.value as Difficulty | "",
                      })
                    }
                    className="w-full appearance-none rounded-lg bg-panel py-2 pl-3 pr-9 text-sm text-ink outline-none transition focus:ring-2 focus:ring-accent/30"
                  >
                    <option value="">Any</option>
                    {difficulties.map((d) => (
                      <option key={d} value={d}>
                        {d}
                      </option>
                    ))}
                  </select>
                  <ChevronDown
                    aria-hidden="true"
                    className="pointer-events-none absolute right-3 top-1/2 h-3 w-3 -translate-y-1/2 text-dim"
                  />
                </div>
              </div>
              <div className="flex flex-1 flex-col gap-1">
                <label className="text-sm font-semibold text-muted">Tag</label>
                <input
                  type="text"
                  list="tag-options"
                  value={draft.tag}
                  onChange={(e) => setDraft({ ...draft, tag: e.target.value })}
                  className="w-full rounded-lg bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:ring-2 focus:ring-accent/30"
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
                className="md:col-span-2 rounded-lg bg-panel px-4 py-2 text-sm font-semibold text-muted transition hover:text-ink"
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
                  className="w-full rounded-xl bg-surface p-5 flex flex-col md:flex-row"
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
                    <Link
                      to={`/assessments/${test.id}/preview`}
                      className="rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light"
                    >
                      Take test
                    </Link>
                  </div>
                </div>
              ))}
            </div>
            <div className="flex items-center justify-between gap-2 mt-5">
              <button
                className="rounded-lg bg-surface px-4 py-2 text-sm font-semibold text-ink transition hover:bg-accent hover:text-shell disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-surface disabled:hover:text-ink"
                disabled={offset <= 0}
                onClick={() => setOffset((prev) => prev - 5)}
              >
                «
              </button>
              <button
                className="rounded-lg bg-surface px-4 py-2 text-sm font-semibold text-ink transition hover:bg-accent hover:text-shell disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-surface disabled:hover:text-ink"
                disabled={offset + 5 >= testCount}
                onClick={() => setOffset((prev) => prev + 5)}
              >
                »
              </button>
            </div>
          </section>

          <aside className="w-full lg:w-[30%]">
            <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
              Updates
            </p>
            <h1 className="mb-6 text-3xl font-bold text-ink">
              Notifications
            </h1>
            <div className="flex flex-col gap-5">
              {notificationsLoading && (
                <div className="w-full rounded-xl bg-surface p-5 text-sm text-muted">
                  Loading notifications…
                </div>
              )}
              {!notificationsLoading && notifications.length === 0 && (
                <div className="w-full rounded-xl bg-surface p-5 text-sm text-muted">
                  No notifications yet.
                </div>
              )}
              {notifications.map((notification) => (
                <div className="w-full rounded-xl bg-surface p-5" key={notification.id}>
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

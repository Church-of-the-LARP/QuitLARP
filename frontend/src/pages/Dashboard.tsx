import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import Alert from "../components/Alert.tsx";
import useAuth from "../scripts/useAuth.tsx";
import client, { getErrorText } from "../scripts/api.ts";
import type {
  ActionResult,
  AssessmentSummary,
  ContinuingSession,
  Notification,
  UserStats,
} from "../types/types.ts";
import usePageTitle from "../scripts/usePageTitle.ts";

const difficultyColor: Record<string, string> = {
  easy: "text-success",
  medium: "text-warning",
  hard: "text-danger",
};

const formatDeadline = (value: string) =>
  new Date(value).toLocaleString(undefined, {
    weekday: "short",
    hour: "2-digit",
    minute: "2-digit",
  });

const formatRelativeTime = (value: string) => {
  const diffMs = Date.now() - new Date(value).getTime();
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
};

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
  const { user } = useAuth();
  const [closeEmail, setCloseEmail] = useState(false);

  const [stats, setStats] = useState<UserStats | null>(null);
  const [sessions, setSessions] = useState<ContinuingSession[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);

  const [assessments, setAssessments] = useState<AssessmentSummary[]>([]);
  const [assessmentsError, setAssessmentsError] = useState<string | null>(
    null,
  );
  const [assessmentsLoading, setAssessmentsLoading] = useState(true);

  const [titleFilter, setTitleFilter] = useState("");
  const [difficultyFilter, setDifficultyFilter] = useState("");
  const [authorFilter, setAuthorFilter] = useState("");
  const [tagFilter, setTagFilter] = useState("");

  useEffect(() => {
    if (!user) return;
    let active = true;
    client
      .GET("/api/v1/users/me/stats")
      .then((res) => {
        if (active && res.data) setStats(res.data);
      })
      .catch(() => {});
    client
      .GET("/api/v1/users/me/sessions")
      .then((res) => {
        if (active && res.data) setSessions(res.data.sessions ?? []);
      })
      .catch(() => {});
    client
      .GET("/api/v1/users/me/notifications")
      .then((res) => {
        if (active && res.data) setNotifications(res.data.notifications ?? []);
      })
      .catch(() => {});
    return () => {
      active = false;
    };
  }, [user]);

  useEffect(() => {
    if (!user) return;
    let active = true;
    setAssessmentsLoading(true);
    const timeout = setTimeout(() => {
      client
        .GET("/api/v1/assessments", {
          params: {
            query: {
              q: titleFilter || undefined,
              difficulty: difficultyFilter || undefined,
              tag: tagFilter || undefined,
              limit: 50,
            },
          },
        })
        .then((res) => {
          if (!active) return;
          if (res.error) {
            setAssessmentsError(
              getErrorText(res.error, "Could not load public tests"),
            );
            return;
          }
          setAssessmentsError(null);
          setAssessments(res.data?.assessments ?? []);
        })
        .catch(() => {
          if (active) setAssessmentsError("Could not load public tests");
        })
        .finally(() => {
          if (active) setAssessmentsLoading(false);
        });
    }, 250);
    return () => {
      active = false;
      clearTimeout(timeout);
    };
  }, [user, titleFilter, difficultyFilter, tagFilter]);

  const markNotificationsRead = () => {
    client
      .POST("/api/v1/users/me/notifications/read")
      .then(() => {
        setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
      })
      .catch(() => {});
  };

  if (!user) return null;

  const visibleAssessments = assessments.filter((a) =>
    (a.author?.username ?? "").toLowerCase().includes(authorFilter.toLowerCase()),
  );
  const unreadCount = notifications.filter((n) => !n.read).length;

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
              <p className="my-2 text-5xl font-bold text-ink">
                {stats?.completed ?? 0}
              </p>
              <p className="text-sm font-medium text-muted">tests</p>
            </div>
            <div className="flex h-full w-full flex-col justify-center rounded-xl border border-line-hover bg-surface px-5 text-center">
              <p className="text-sm font-semibold uppercase text-muted">
                In progress
              </p>
              <p className="my-2 text-5xl font-bold text-ink">
                {stats?.inProgress ?? 0}
              </p>
              <p className="text-sm font-medium text-muted">tests</p>
            </div>
            <div className="flex h-full w-full flex-col justify-center rounded-xl border border-line-hover bg-surface px-5 text-center">
              <p className="text-sm font-semibold uppercase text-muted">
                Completion rate
              </p>
              <p className="my-2 text-5xl font-bold text-ink">
                {stats?.completionRate ?? 0}%
              </p>
              <p className="text-sm font-medium text-muted">of started tests</p>
            </div>
          </div>
        </section>

        <section className="my-6">
          <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
            Keep learning
          </p>
          <h1 className="mb-6 text-3xl font-bold text-ink">
            Continue solving
          </h1>
          {sessions.length === 0 ? (
            <p className="text-sm text-muted">
              Nothing in progress. Start a public test below to see it here.
            </p>
          ) : (
            <div className="flex flex-nowrap overflow-auto justify-start items-center gap-5">
              {sessions.map((session) => (
                <div
                  className="flex flex-col w-96 shrink-0 rounded-xl border border-line-hover bg-surface p-5 md:flex-row"
                  key={session.assessmentId}
                >
                  <div className="w-full">
                    <h2 className="text-lg font-semibold text-ink">
                      {session.assessmentTitle}
                    </h2>
                    <p className="mt-2 text-sm font-medium text-muted">
                      {session.chapterTitle}
                    </p>
                    <p className="mt-1 text-sm text-muted">
                      Ends {formatDeadline(session.deadline)}
                    </p>
                  </div>
                  <div className="w-full md:w-40 text-center content-center md:content-end md:text-end">
                    <Link
                      to={`/assessments/${session.assessmentId}/solve`}
                      className="inline-block rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light"
                    >
                      Continue
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>

        <div className="flex gap-5">
          <section className="w-[70%]">
            <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
              Explore
            </p>
            <h1 className="mb-5 text-3xl font-bold text-ink">Public tests</h1>
            <div className="flex flex-row gap-3 mb-2">
              <div className="flex flex-col gap-1 w-full">
                <label className="text-sm font-semibold text-muted">
                  Search public tests
                </label>
                <input
                  type="text"
                  value={titleFilter}
                  onChange={(e) => setTitleFilter(e.target.value)}
                  className="mb-3 rounded-lg border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30"
                  placeholder="Search for title"
                />
              </div>
              <div className="flex flex-col gap-1 w-full">
                <label className="text-sm font-semibold text-muted">
                  Filter by difficulty
                </label>
                <select
                  value={difficultyFilter}
                  className="mb-3 rounded-lg border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30"
                  onChange={(e) => setDifficultyFilter(e.target.value)}
                >
                  <option value="">All</option>
                  <option value="easy">Easy</option>
                  <option value="medium">Medium</option>
                  <option value="hard">Hard</option>
                </select>
              </div>
              <div className="flex flex-col gap-1 w-full">
                <label className="text-sm font-semibold text-muted">
                  Filter by author
                </label>
                <input
                  type="text"
                  value={authorFilter}
                  onChange={(e) => setAuthorFilter(e.target.value)}
                  className="mb-3 rounded-lg border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30"
                  placeholder="Search for author"
                />
              </div>
              <div className="flex flex-col gap-1 w-full">
                <label className="text-sm font-semibold text-muted">
                  Filter by tag
                </label>
                <input
                  type="text"
                  value={tagFilter}
                  onChange={(e) => setTagFilter(e.target.value)}
                  className="mb-3 rounded-lg border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30"
                  placeholder="Search for tag"
                />
              </div>
            </div>

            {assessmentsError && <Alert kind="error" text={assessmentsError} />}

            <div className="flex flex-col gap-5">
              {!assessmentsLoading &&
                !assessmentsError &&
                visibleAssessments.length === 0 && (
                  <p className="text-sm text-muted">
                    No public tests match these filters.
                  </p>
                )}
              {visibleAssessments.map((test) => (
                <div
                  className="w-full rounded-xl border border-line-hover bg-surface p-5 flex flex-col md:flex-row"
                  key={test.id}
                >
                  <div className="w-full">
                    <h2 className="text-lg font-semibold text-ink">
                      {test.title}
                    </h2>
                    <p className="mt-2 text-sm text-muted">
                      By {test.author?.username ?? "unknown"}
                    </p>
                    <p
                      className={`mt-1 text-xs font-semibold uppercase ${difficultyColor[test.difficulty] ?? "text-muted"}`}
                    >
                      {test.difficulty}
                    </p>
                    {test.tags && test.tags.length > 0 && (
                      <p className="mt-1 text-xs text-dim">
                        {test.tags.map((tag) => tag.name).join(", ")}
                      </p>
                    )}
                  </div>
                  <div className="w-full md:w-30 text-center content-center md:content-end md:text-end">
                    <Link
                      to={`/assessments/${test.id}/preview`}
                      className="inline-block rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light"
                    >
                      Take test
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          </section>

          <aside className="w-[30%]">
            <div className="mb-2 flex items-center justify-between">
              <p className="text-xs font-semibold uppercase text-accent-light">
                Updates
              </p>
              {unreadCount > 0 && (
                <button
                  onClick={markNotificationsRead}
                  className="text-xs font-semibold text-accent-light hover:text-accent-lighter"
                >
                  Mark all read
                </button>
              )}
            </div>
            <h1 className="mb-6 text-3xl font-bold text-ink">Notifications</h1>
            {notifications.length === 0 ? (
              <p className="text-sm text-muted">Nothing yet.</p>
            ) : (
              <div className="flex flex-col gap-5">
                {notifications.map((notification) => (
                  <div
                    className={`w-full rounded-xl border border-line-hover bg-surface p-5 ${notification.read ? "" : "border-accent-light"}`}
                    key={notification.id}
                  >
                    <h2 className="text-base font-semibold text-ink">
                      {notification.title}
                    </h2>
                    <p className="mt-2 text-sm leading-5 text-muted">
                      {notification.detail}
                    </p>
                    <p className="mt-3 text-xs font-medium uppercase text-dim">
                      {formatRelativeTime(notification.createdAt)}
                    </p>
                  </div>
                ))}
              </div>
            )}
          </aside>
        </div>
      </main>
    </>
  );
}

export default App;

import { useState } from "react";
import Alert from "../components/Alert.tsx";
import useAuth from "../scripts/useAuth.tsx";
import type { ActionResult } from "../types/types.ts";

const upcomingTests = [
  { title: "Intro to LARP Safety", date: "Today, 18:00", questions: 12 },
  { title: "Character Development", date: "Tomorrow, 10:30", questions: 8 },
  { title: "Worldbuilding Basics", date: "Friday, 14:00", questions: 15 },
  { title: "Intro to LARP Safety", date: "Today, 18:00", questions: 12 },
  { title: "Character Development", date: "Tomorrow, 10:30", questions: 8 },
  { title: "Worldbuilding Basics", date: "Friday, 14:00", questions: 15 },
  { title: "Intro to LARP Safety", date: "Today, 18:00", questions: 12 },
  { title: "Character Development", date: "Tomorrow, 10:30", questions: 8 },
  { title: "Worldbuilding Basics", date: "Friday, 14:00", questions: 15 },
  { title: "Intro to LARP Safety", date: "Today, 18:00", questions: 12 },
  { title: "Character Development", date: "Tomorrow, 10:30", questions: 8 },
  { title: "Worldbuilding Basics", date: "Friday, 14:00", questions: 15 },
];

const publicTests = [
  {
    title: "Improvisation Essentials",
    author: "Mara Chen",
    difficulty: "Beginner",
    attempts: 248,
  },
  {
    title: "Designing Memorable Factions",
    author: "Jon Bell",
    difficulty: "Intermediate",
    attempts: 96,
  },
  {
    title: "Collaborative Storytelling",
    author: "Priya Shah",
    difficulty: "Beginner",
    attempts: 184,
  },
];

const difficultyColor: Record<string, string> = {
  Beginner: "text-success",
  Intermediate: "text-warning",
  Advanced: "text-danger",
};

const notifications = [
  {
    title: "Your test was published",
    detail: "Improvisation Essentials is now public",
    time: "2h ago",
  },
  {
    title: "New comment on your answer",
    detail: "Mara Chen replied to your discussion",
    time: "Yesterday",
  },
  {
    title: "You earned a new badge",
    detail: "Completed 5 tests",
    time: "3d ago",
  },
];

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
    <div className="mb-6 rounded-md border border-accent bg-surface px-4 py-3 text-sm text-accent">
      <p className="font-medium">Verify your email address</p>
      <p className="mt-0.5 text-accent">
        We emailed a verification link to{" "}
        <span className="font-semibold">{user?.email}</span>. Didn't get it?{" "}
        <button
          onClick={() => handleResend()}
          className="font-semibold underline underline-offset-2 hover:text-accent-light"
        >
          Resend the email
        </button>
        .
      </p>
      <button
        className="absolute top-1 right-3 p-2 hover:text-accent-light"
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
  const { user } = useAuth();
  const [closeEmail, setCloseEmail] = useState(false);
  const [filter, setFilter] = useState("");

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
          <h1 className="mb-6 text-3xl font-bold text-ink">
            Upcoming tests
          </h1>
          <div className="flex flex-nowrap overflow-auto justify-start items-center gap-5">
            {upcomingTests.map((test, idx) => (
              <div
                className="flex flex-col w-96 shrink-0 rounded-xl border border-line-hover bg-surface p-5 md:flex-row"
                key={idx}
              >
                <div className="w-full">
                  <h2 className="text-lg font-semibold text-ink">
                    {test.title}
                  </h2>
                  <p className="mt-2 text-sm font-medium text-muted">
                    {test.date}
                  </p>
                  <p className="mt-1 text-sm text-muted">
                    {test.questions} questions
                  </p>
                </div>
                <div className="w-full md:w-40 text-center content-center md:content-end md:text-end">
                  <button className="rounded-lg bg-accent px-4 py-2 text-sm font-semibold text-shell transition hover:bg-accent-light">
                    Take test
                  </button>
                </div>
              </div>
            ))}
          </div>
        </section>

        <div className="flex gap-5">
          <section className="w-[70%]">
            <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
              Explore
            </p>
            <h1 className="mb-5 text-3xl font-bold text-ink">
              Public tests
            </h1>
            <div className="flex flex-col gap-1 mb-2">
              <label className="text-sm font-semibold text-muted">
                Search public tests
              </label>
              <input
                type="text"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                className="mb-3 w-90 rounded-lg border border-line-hover bg-panel px-3 py-2 text-sm text-ink outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/30"
                placeholder="Search for title"
              />
            </div>

            <div className="flex flex-col gap-5">
              {publicTests
                .filter((e) =>
                  e.title.toUpperCase().includes(filter.toUpperCase()),
                )
                .map((test, idx) => (
                  <div
                    className="w-full rounded-xl border border-line-hover bg-surface p-5 flex flex-col md:flex-row"
                    key={idx}
                  >
                    <div className="w-full">
                      <h2 className="text-lg font-semibold text-ink">
                        {test.title}
                      </h2>
                      <p className="mt-2 text-sm text-muted">
                        By {test.author}
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
          </section>

          <aside className="w-[30%]">
            <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
              Updates
            </p>
            <h1 className="mb-6 text-3xl font-bold text-ink">
              Notifications
            </h1>
            <div className="flex flex-col gap-5">
              {notifications.map((notification, idx) => (
                <div
                  className="w-full rounded-xl border border-line-hover bg-surface p-5"
                  key={idx}
                >
                  <h2 className="text-base font-semibold text-ink">
                    {notification.title}
                  </h2>
                  <p className="mt-2 text-sm leading-5 text-muted">
                    {notification.detail}
                  </p>
                  <p className="mt-3 text-xs font-medium uppercase text-dim">
                    {notification.time}
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

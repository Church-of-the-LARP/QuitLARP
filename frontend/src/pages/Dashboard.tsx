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
    <div className="mb-6 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
      <p className="font-medium">Verify your email address</p>
      <p className="mt-0.5 text-amber-700">
        We emailed a verification link to{" "}
        <span className="font-semibold">{user?.email}</span>. Didn't get it?{" "}
        <button
          onClick={() => handleResend()}
          className="font-semibold underline underline-offset-2 hover:text-amber-900"
        >
          Resend the email
        </button>
        .
      </p>
      <button
        className="absolute top-3 right-5"
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
          <h1 className="font-bold text-3xl mb-6">Upcoming tests</h1>
          <div className="flex flex-nowrap overflow-auto justify-start items-center gap-5">
            {upcomingTests.map((test, idx) => (
              <div
                className="flex flex-col w-96 shrink-0 rounded-xl border p-5 md:flex-row"
                key={idx}
              >
                <div className="w-full">
                  <h2>{test.title}</h2>
                  <p>{test.date}</p>
                  <p>{test.questions} questions</p>
                </div>
                <div className="w-full md:w-30 text-center content-center md:content-end md:text-end">
                  <button className="text-white p-2 bg-blue-500 rounded-xl">
                    Take test
                  </button>
                </div>
              </div>
            ))}
          </div>
        </section>

        <div className="flex gap-5">
          <section className="w-[70%]">
            <h1 className="font-bold text-3xl mb-3">Public tests</h1>
            <div className="flex flex-col gap-1 mb-2">
              <label className="font-semibold">Search</label>
              <input
                type="text"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                className="w-90 px-3 py-2 border rounded-xl mb-3"
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
                    className="w-full rounded-xl border p-5 flex flex-col md:flex-row"
                    key={idx}
                  >
                    <div className="w-full">
                      <h2>{test.title}</h2>
                      <p>{test.author}</p>
                      <p>{test.difficulty}</p>
                    </div>
                    <div className="w-full md:w-30 text-center content-center md:content-end md:text-end">
                      <button className="text-white p-2 bg-blue-500 rounded-xl">
                        Take test
                      </button>
                    </div>
                  </div>
                ))}
            </div>
          </section>

          <aside className="w-[30%]">
            <h1 className="font-bold text-3xl mb-6">Notifications</h1>
            <div className="flex flex-col gap-5">
              {notifications.map((notification, idx) => (
                <div className="w-full rounded-xl border p-5" key={idx}>
                  <h2>{notification.title}</h2>
                  <p>{notification.detail}</p>
                  <p>{notification.time}</p>
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

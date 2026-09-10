import { useState } from "react";
import Alert from "../components/Alert.tsx";
import { Link } from "react-router-dom";
import useAuth from "../scripts/useAuth.tsx";
import type { ActionResult, Role } from "../types/types.ts";

const roleStyles: Record<Role, string> = {
  user: "bg-gray-100 text-gray-700",
  admin: "bg-violet-100 text-violet-700",
  superadmin: "bg-amber-100 text-amber-800",
};

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

function VerifyEmailBanner() {
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
      {alertMsg.error && <Alert kind="error" text={alertMsg.error} />}
      {alertMsg.notice && <Alert kind="notice" text={alertMsg.notice} />}
    </div>
  );
}

function App() {
  const { user, logoutUser } = useAuth();

  const handleLogout = async () => {
    await logoutUser();
  };

  if (!user) return null;

  const isAdmin = user.role === "admin" || user.role === "superadmin";

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-6 py-3">
          <div className="flex items-center gap-4">
            <Link to="/" className="text-sm font-semibold text-gray-900">
              QuitLARP
            </Link>
            {isAdmin && (
              <Link
                to="/users"
                className="text-sm text-gray-500 hover:text-gray-800"
              >
                Users
              </Link>
            )}
          </div>
          <div className="flex items-center gap-3">
            <span className="hidden text-sm text-gray-600 sm:inline">
              {user.username}
              <span className="mx-1.5 text-gray-300">·</span>
              {user.email}
            </span>
            <span
              className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[user.role]}`}
            >
              {user.role}
            </span>
            <button
              onClick={() => handleLogout()}
              className="rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100"
            >
              Sign out
            </button>
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-3xl py-6">
        {!user.emailVerified && <VerifyEmailBanner />}
      </div>
      <main className="mx-auto w-[80vw] flex flex-col gap-5">
        <section className="mb-6">
          <h1 className="bold text-2xl mb-6">Upcoming tests</h1>
          <div className="flex flex-nowrap overflow-auto justify-start items-center gap-5">
            {upcomingTests.map((test) => (
              <div
                className="flex flex-col w-96 shrink-0 rounded-xl border p-5 md:flex-row"
                key={test.title}
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
            <h1 className="bold text-2xl mb-6">Public tests</h1>
            <div className="flex flex-col gap-5">
              {publicTests.map((test) => (
                <div
                  className="w-full rounded-xl border p-5 flex flex-col md:flex-row"
                  key={test.title}
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
            <h1 className="bold text-2xl mb-6">Notifications</h1>
            <div className="flex flex-col gap-5">
              {notifications.map((notification) => (
                <div
                  className="w-full rounded-xl border p-5"
                  key={notification.title}
                >
                  <h2>{notification.title}</h2>
                  <p>{notification.detail}</p>
                  <p>{notification.time}</p>
                </div>
              ))}
            </div>
          </aside>
        </div>
      </main>
    </div>
  );
}

export default App;

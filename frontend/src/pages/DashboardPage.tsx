import { useState } from "react";
import { Link } from "react-router-dom";
import { useAuth, type ActionResult, type Role } from "../auth/AuthContext";
import { canViewUsers } from "./UsersPage";
import { Notice } from "../components/Notice";

const roleStyles: Record<Role, string> = {
  user: "bg-gray-100 text-gray-700",
  admin: "bg-violet-100 text-violet-700",
  superadmin: "bg-amber-100 text-amber-800",
};

function VerifyEmailBanner() {
  const { user, resendVerification } = useAuth();
  const [result, setResult] = useState<ActionResult>({});

  async function resend() {
    setResult(await resendVerification());
  }

  return (
    <div className="mb-6 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
      <p className="font-medium">Verify your email address</p>
      <p className="mt-0.5 text-amber-700">
        We emailed a verification link to{" "}
        <span className="font-semibold">{user?.email}</span>. Didn't get it?{" "}
        <button
          onClick={() => void resend()}
          className="font-semibold underline underline-offset-2 hover:text-amber-900"
        >
          Resend the email
        </button>
        .
      </p>
      {result.error && <Notice kind="error">{result.error}</Notice>}
      {result.notice && <Notice kind="notice">{result.notice}</Notice>}
    </div>
  );
}

export function DashboardPage() {
  const { user, signOut } = useAuth();

  if (!user) {
    return null;
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-6 py-3">
          <div className="flex items-center gap-4">
            <Link to="/" className="text-sm font-semibold text-gray-900">
              QuitLARP
            </Link>
            {canViewUsers(user.role) && (
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
              onClick={() => void signOut()}
              className="rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100"
            >
              Sign out
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-6 py-8">
        {!user.emailVerified && <VerifyEmailBanner />}
        <div className="mb-6 rounded-md border border-gray-200 bg-white px-5 py-4">
          <p className="text-xs font-medium uppercase tracking-wide text-gray-400">
            Signed in as
          </p>
          <p className="mt-1 text-sm text-gray-800">
            <span className="font-semibold">{user.username}</span>
            {user.googleLinked && (
              <span className="ml-2 rounded bg-blue-50 px-1.5 py-0.5 text-xs text-blue-700">
                google account
              </span>
            )}
          </p>
        </div>
      </main>
    </div>
  );
}

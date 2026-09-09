import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  useAuth,
  type ActionResult,
  type Role,
  type User,
} from "../auth/AuthContext";
import { client, errorMessage } from "../api/client";
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

export const canViewUsers = (role: Role | undefined) =>
  role === "admin" || role === "superadmin";

export function UsersPage() {
  const { user, signOut } = useAuth();
  const admin = canViewUsers(user?.role);
  const superadmin = user?.role === "superadmin";

  const [users, setUsers] = useState<User[]>([]);
  const [message, setMessage] = useState<{
    kind: "error" | "notice";
    text: string;
  } | null>(null);
  const [busyId, setBusyId] = useState<number | null>(null);

  const load = useCallback(async () => {
    const res = await client.GET("/api/v1/users");
    if (res.error) {
      setUsers([]);
      setMessage({
        kind: "error",
        text: errorMessage(res.error, "Could not load the user list"),
      });
      return;
    }
    setMessage(null);
    setUsers(res.data?.users ?? []);
  }, []);

  useEffect(() => {
    if (admin) {
      void load();
    }
  }, [admin, load]);

  async function changeRole(u: User, role: Role) {
    setBusyId(u.id);
    setMessage(null);
    const res = await client.PATCH("/api/v1/users/{id}/role", {
      params: { path: { id: u.id } },
      body: { role },
    });
    setBusyId(null);
    if (res.error) {
      setMessage({
        kind: "error",
        text: errorMessage(res.error, "Could not change the role"),
      });
      return;
    }
    await load();
  }

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
        <div className="space-y-8">
          {message && <Notice kind={message.kind}>{message.text}</Notice>}

          {admin && (
            <section>
              <div className="flex items-baseline justify-between">
                <h2 className="text-lg font-semibold">Users</h2>
                <span className="text-xs text-gray-400">
                  {users.length} total
                </span>
              </div>
              <ul className="mt-3 divide-y divide-gray-100 rounded-md border border-gray-200">
                {users.map((u) => (
                  <li
                    key={u.id}
                    className="flex items-center justify-between gap-3 px-4 py-3"
                  >
                    <div className="min-w-0">
                      <p className="flex items-center gap-2 text-sm font-medium text-gray-900">
                        <span className="truncate">{u.username}</span>
                        <span
                          className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[user.role]}`}
                        >
                          {user.role}
                        </span>
                      </p>
                      <p className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-gray-500">
                        <span className="truncate">{u.email}</span>
                        <span>
                          {u.emailVerified ? "✓ verified" : "unverified"}
                        </span>
                        {u.googleLinked && <span>Google account</span>}
                      </p>
                    </div>
                    {superadmin && u.role !== "superadmin" && (
                      <div className="flex shrink-0 gap-2">
                        {u.role === "user" ? (
                          <button
                            onClick={() => void changeRole(u, "admin")}
                            disabled={busyId === u.id}
                            className="rounded-md border border-violet-200 px-2.5 py-1 text-xs font-medium text-violet-700 hover:bg-violet-50 disabled:opacity-50"
                          >
                            Make admin
                          </button>
                        ) : (
                          <button
                            onClick={() => void changeRole(u, "user")}
                            disabled={busyId === u.id}
                            className="rounded-md border border-gray-200 px-2.5 py-1 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50"
                          >
                            Revoke admin
                          </button>
                        )}
                      </div>
                    )}
                  </li>
                ))}
                {users.length === 0 && (
                  <li className="px-4 py-8 text-center text-sm text-gray-400">
                    No users yet
                  </li>
                )}
              </ul>
            </section>
          )}
        </div>
      </main>
    </div>
  );
}

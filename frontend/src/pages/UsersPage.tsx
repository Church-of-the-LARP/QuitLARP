import { useCallback, useEffect, useState } from "react";
import { useAuth, type Role, type User } from "../auth/AuthContext";
import { client, errorMessage } from "../api/client";
import { Notice, RoleBadge } from "../components/ui";

export const canViewUsers = (role: Role | undefined) =>
  role === "admin" || role === "superadmin";

export function UsersPage() {
  const { user } = useAuth();
  const admin = canViewUsers(user?.role);
  const superadmin = user?.role === "superadmin";

  const [users, setUsers] = useState<User[]>([]);
  const [message, setMessage] = useState<{ kind: "error" | "notice"; text: string } | null>(null);
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
      setMessage({ kind: "error", text: errorMessage(res.error, "Could not change the role") });
      return;
    }
    await load();
  }

  return (
    <div className="space-y-8">
      {message && <Notice kind={message.kind}>{message.text}</Notice>}

      {admin && (
        <section>
          <div className="flex items-baseline justify-between">
            <h2 className="text-lg font-semibold">Users</h2>
            <span className="text-xs text-gray-400">{users.length} total</span>
          </div>
          <ul className="mt-3 divide-y divide-gray-100 rounded-md border border-gray-200">
            {users.map((u) => (
              <li key={u.id} className="flex items-center justify-between gap-3 px-4 py-3">
                <div className="min-w-0">
                  <p className="flex items-center gap-2 text-sm font-medium text-gray-900">
                    <span className="truncate">{u.username}</span>
                    <RoleBadge role={u.role} />
                  </p>
                  <p className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-gray-500">
                    <span className="truncate">{u.email}</span>
                    <span>{u.emailVerified ? "✓ verified" : "unverified"}</span>
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
              <li className="px-4 py-8 text-center text-sm text-gray-400">No users yet</li>
            )}
          </ul>
        </section>
      )}
    </div>
  );
}

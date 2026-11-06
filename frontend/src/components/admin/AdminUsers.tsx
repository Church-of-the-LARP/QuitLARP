import { useCallback, useEffect, useState } from 'react';
import Alert from '../Alert.tsx';
import useAuth from '../../scripts/useAuth.tsx';
import client, { getErrorText } from '../../scripts/api.ts';
import type { AlertMessage, Role, User } from '../../types/types.ts';

const roleStyles: Record<Role, string> = {
  user: 'bg-raised text-muted',
  moderator: 'bg-accent-secondary/20 text-accent-secondary',
  admin: 'bg-warning/20 text-warning',
};
const roles: Role[] = ['user', 'moderator', 'admin'];

function isRole(value: string): value is Role {
  return value === 'user' || value === 'moderator' || value === 'admin';
}

export default function AdminUsers() {
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [users, setUsers] = useState<User[]>([]);
  const [alertMsg, setAlertMsg] = useState<AlertMessage | null>(null);
  const [loadingId, setLoadingId] = useState<number | null>(null);

  const loadUsers = useCallback(async () => {
    const res = await client.GET('/api/v1/users');
    if (res.error) {
      setUsers([]);
      setAlertMsg({
        kind: 'error',
        text: getErrorText(res.error, 'Could not load the user list'),
      });
      return;
    }

    setAlertMsg(null);
    setUsers(res.data?.users ?? []);
  }, []);

  useEffect(() => {
    if (isAdmin) void loadUsers();
  }, [isAdmin, loadUsers]);

  const handleRoleChange = async (target: User, role: Role) => {
    if (!user) return;
    setLoadingId(target.id);
    setAlertMsg(null);
    try {
      const res = await client.PATCH('/api/v1/users/{id}/role', {
        params: { path: { id: target.id } },
        body: { role },
      });
      if (res.error) {
        setAlertMsg({
          kind: 'error',
          text: getErrorText(res.error, 'Could not change the role'),
        });
        return;
      }

      await loadUsers();
    } catch (error) {
      setAlertMsg({
        kind: 'error',
        text:
          error instanceof Error ? error.message : 'Could not change the role',
      });
    } finally {
      setLoadingId(null);
    }
  };

  if (!user || !isAdmin) return null;

  return (
    <section>
      {alertMsg && (
        <div className="mb-4">
          <Alert kind={alertMsg.kind} text={alertMsg.text} />
        </div>
      )}
      <div className="flex items-baseline justify-between">
        <div>
          <h2 className="text-xl font-semibold text-ink">Users</h2>
          <p className="mt-1 text-sm text-muted">
            Manage accounts and assign user, moderator, or admin roles.
          </p>
        </div>
        <span className="text-xs text-dim">{users.length} total</span>
      </div>
      <ul className="mt-3 rounded-md bg-surface">
        {users.map((target) => (
          <li
            key={target.id}
            className="flex items-center justify-between gap-3 px-4 py-3"
          >
            <div className="min-w-0">
              <p className="flex items-center gap-2 text-sm font-medium text-ink">
                <span className="truncate">{target.username}</span>
                <span
                  className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[target.role]}`}
                >
                  {target.role}
                </span>
              </p>
              <p className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-muted">
                <span className="truncate">{target.email}</span>
                <span>
                  {target.emailVerified ? '✓ verified' : 'unverified'}
                </span>
                {target.googleLinked && <span>Google account</span>}
              </p>
            </div>
            <div className="relative shrink-0">
              <select
                value={target.role}
                disabled={loadingId === target.id || target.id === user.id}
                title={
                  target.id === user.id
                    ? "You can't change your own role"
                    : undefined
                }
                onChange={(event) => {
                  const role = event.target.value;
                  if (isRole(role)) void handleRoleChange(target, role);
                }}
                aria-label={`Role for ${target.username}`}
                className="scheme-dark appearance-none rounded-md bg-panel py-1 pl-2.5 pr-8 text-xs font-medium text-ink transition-colors hover:bg-raised focus:bg-raised focus:outline-none focus:ring-2 focus:ring-accent/30 disabled:opacity-50"
              >
                {roles.map((role) => (
                  <option key={role} value={role} className="bg-panel text-ink">
                    {role}
                  </option>
                ))}
              </select>
              <svg
                aria-hidden="true"
                viewBox="0 0 16 16"
                fill="none"
                className="pointer-events-none absolute right-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted"
              >
                <path
                  d="m4 6 4 4 4-4"
                  stroke="currentColor"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="1.5"
                />
              </svg>
            </div>
          </li>
        ))}
        {users.length === 0 && !alertMsg && (
          <li className="px-4 py-8 text-center text-sm text-dim">
            No users yet
          </li>
        )}
      </ul>
    </section>
  );
}

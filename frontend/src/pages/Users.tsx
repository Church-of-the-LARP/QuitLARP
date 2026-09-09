import { useCallback, useEffect, useState } from 'react';
import Alert from '../components/Alert.tsx';
import { Link } from 'react-router-dom';
import useAuth from '../scripts/useAuth.tsx';
import client, { getErrorText } from '../scripts/api.ts';
import type {
  ActionResult,
  AlertMessage,
  Role,
  User,
} from '../types/types.ts';

const roleStyles: Record<Role, string> = {
  user: 'bg-gray-100 text-gray-700',
  admin: 'bg-violet-100 text-violet-700',
  superadmin: 'bg-amber-100 text-amber-800',
};

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
        We emailed a verification link to{' '}
        <span className="font-semibold">{user?.email}</span>. Didn't get it?{' '}
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
  const isAdmin = user?.role === 'admin' || user?.role === 'superadmin';
  const isSuperadmin = user?.role === 'superadmin';

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
    if (isAdmin) loadUsers();
  }, [isAdmin, loadUsers]);

  const handleLogout = async () => {
    await logoutUser();
  };

  const handleRoleChange = async (target: User, role: Role) => {
    setLoadingId(target.id);
    setAlertMsg(null);
    const res = await client.PATCH('/api/v1/users/{id}/role', {
      params: { path: { id: target.id } },
      body: { role },
    });
    setLoadingId(null);

    if (res.error) {
      setAlertMsg({
        kind: 'error',
        text: getErrorText(res.error, 'Could not change the role'),
      });
      return;
    }

    await loadUsers();
  };

  if (!user) return null;

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

      <main className="mx-auto max-w-3xl px-6 py-8">
        {!user.emailVerified && <VerifyEmailBanner />}
        <div className="space-y-8">
          {alertMsg && <Alert kind={alertMsg.kind} text={alertMsg.text} />}

          {isAdmin && (
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
                          className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[u.role]}`}
                        >
                          {u.role}
                        </span>
                      </p>
                      <p className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-gray-500">
                        <span className="truncate">{u.email}</span>
                        <span>
                          {u.emailVerified ? '✓ verified' : 'unverified'}
                        </span>
                        {u.googleLinked && <span>Google account</span>}
                      </p>
                    </div>
                    {isSuperadmin && u.role !== 'superadmin' && (
                      <div className="flex shrink-0 gap-2">
                        {u.role === 'user' ? (
                          <button
                            onClick={() => handleRoleChange(u, 'admin')}
                            disabled={loadingId === u.id}
                            className="rounded-md border border-violet-200 px-2.5 py-1 text-xs font-medium text-violet-700 hover:bg-violet-50 disabled:opacity-50"
                          >
                            Make admin
                          </button>
                        ) : (
                          <button
                            onClick={() => handleRoleChange(u, 'user')}
                            disabled={loadingId === u.id}
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

export default App;

import { useCallback, useEffect, useState } from 'react';
import Alert from '../components/Alert.tsx';
import useAuth from '../scripts/useAuth.tsx';
import client, { getErrorText } from '../scripts/api.ts';
import type {
  ActionResult,
  AlertMessage,
  Role,
  User,
} from '../types/types.ts';

const roleStyles: Record<Role, string> = {
  user: 'bg-zinc-700 text-zinc-300',
  admin: 'bg-teal-400/20 text-teal-400',
  superadmin: 'bg-amber-400/20 text-amber-400',
};

function VerifyEmailBanner() {
  const { user, resendVerification } = useAuth();
  const [alertMsg, setAlertMsg] = useState<ActionResult>({});

  const handleResend = async () => {
    setAlertMsg(await resendVerification());
  };

  return (
    <div className="mb-6 rounded-md border border-amber-400/30 bg-amber-400/10 px-4 py-3 text-sm text-amber-400">
      <p className="font-medium">Verify your email address</p>
      <p className="mt-0.5 text-amber-400/80">
        We emailed a verification link to{' '}
        <span className="font-semibold">{user?.email}</span>. Didn't get it?{' '}
        <button
          onClick={() => handleResend()}
          className="font-semibold underline underline-offset-2 hover:text-amber-300"
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
  const { user } = useAuth();
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
    <main className="mx-auto max-w-3xl px-6 py-8">
      {!user.emailVerified && <VerifyEmailBanner />}
      <div className="space-y-8">
        {alertMsg && <Alert kind={alertMsg.kind} text={alertMsg.text} />}

        {isAdmin && (
          <section>
            <div className="flex items-baseline justify-between">
              <h2 className="text-lg font-semibold text-zinc-100">Users</h2>
              <span className="text-xs text-zinc-500">
                {users.length} total
              </span>
            </div>
            <ul className="mt-3 divide-y divide-zinc-800 rounded-md border border-zinc-700 bg-zinc-800">
              {users.map((u) => (
                <li
                  key={u.id}
                  className="flex items-center justify-between gap-3 px-4 py-3"
                >
                  <div className="min-w-0">
                    <p className="flex items-center gap-2 text-sm font-medium text-zinc-100">
                      <span className="truncate">{u.username}</span>
                      <span
                        className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[u.role]}`}
                      >
                        {u.role}
                      </span>
                    </p>
                    <p className="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-zinc-400">
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
                          className="rounded-md border border-teal-400/30 px-2.5 py-1 text-xs font-medium text-teal-400 hover:bg-teal-400/10 disabled:opacity-50"
                        >
                          Make admin
                        </button>
                      ) : (
                        <button
                          onClick={() => handleRoleChange(u, 'user')}
                          disabled={loadingId === u.id}
                          className="rounded-md border border-zinc-700 px-2.5 py-1 text-xs font-medium text-zinc-400 hover:bg-zinc-700 disabled:opacity-50"
                        >
                          Revoke admin
                        </button>
                      )}
                    </div>
                  )}
                </li>
              ))}
              {users.length === 0 && (
                <li className="px-4 py-8 text-center text-sm text-zinc-500">
                  No users yet
                </li>
              )}
            </ul>
          </section>
        )}
      </div>
    </main>
  );
}

export default App;

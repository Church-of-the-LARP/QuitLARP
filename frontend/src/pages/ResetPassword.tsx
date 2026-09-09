import { useState, type FormEvent } from 'react';
import Alert from '../components/Alert.tsx';
import { Link, useSearchParams } from 'react-router-dom';
import client, { getErrorText } from '../scripts/api.ts';
import type { AlertMessage } from '../types/types.ts';

function App() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';
  const [newPass, setNewPass] = useState('');
  const [confirmPass, setConfirmPass] = useState('');
  const [alertMsg, setAlertMsg] = useState<AlertMessage | null>(null);
  const [loading, setLoading] = useState(false);

  const handleReset = async (e: FormEvent) => {
    e.preventDefault();
    setAlertMsg(null);

    if (newPass.length < 8) {
      setAlertMsg({
        kind: 'error',
        text: 'Password must be at least 8 characters',
      });
      return;
    }
    if (newPass !== confirmPass) {
      setAlertMsg({ kind: 'error', text: 'Passwords do not match' });
      return;
    }

    setLoading(true);
    try {
      const res = await client.POST('/api/v1/auth/reset-password', {
        body: { token, newPassword: newPass },
      });
      setAlertMsg(
        res.error
          ? {
              kind: 'error',
              text: getErrorText(res.error, 'Could not reset the password'),
            }
          : { kind: 'notice', text: res.data?.message ?? 'Password updated.' },
      );
    } catch (err) {
      console.error('Reset password error:', err);
      setAlertMsg({ kind: 'error', text: 'Could not reset the password' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto w-full max-w-sm">
      <h1 className="text-2xl font-semibold">Reset your password</h1>
      <p className="mt-1 text-sm text-gray-500">
        Choose a new password for your account
      </p>

      {alertMsg && (
        <div className="mt-4">
          <Alert kind={alertMsg.kind} text={alertMsg.text} />
        </div>
      )}

      {alertMsg?.kind === 'notice' ? (
        <Link
          to="/login"
          className="mt-6 block w-full rounded-md bg-blue-600 px-4 py-2 text-center text-sm font-medium text-white hover:bg-blue-700"
        >
          Go to sign in
        </Link>
      ) : (
        <form onSubmit={handleReset} className="mt-6 space-y-4">
          <label className="block">
            <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500">
              New password
            </span>
            <input
              type="password"
              required
              autoComplete="new-password"
              value={newPass}
              onChange={(e) => setNewPass(e.target.value)}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500">
              Repeat new password
            </span>
            <input
              type="password"
              required
              autoComplete="new-password"
              value={confirmPass}
              onChange={(e) => setConfirmPass(e.target.value)}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
          </label>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
          >
            {loading ? 'Updating…' : 'Update password'}
          </button>
        </form>
      )}
    </div>
  );
}

export default App;

import { useState, type FormEvent } from 'react';
import Alert from '../components/Alert.tsx';
import { Link, useNavigate } from 'react-router-dom';
import client, { getErrorText } from '../scripts/api.ts';

function App() {
  const navigate = useNavigate();
  const [resetEmail, setResetEmail] = useState('');
  const [alertMsg, setAlertMsg] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setAlertMsg(null);
    try {
      const res = await client.POST('/api/v1/auth/forgot-password', {
        body: { email: resetEmail.trim() },
      });
      if (res.error) {
        setAlertMsg(getErrorText(res.error, 'Could not send the reset link'));
        return;
      }

      navigate('/login', {
        state: {
          notice:
            res.data?.message ??
            'If an account exists for that email, a reset link is on its way.',
        },
      });
    } catch (err) {
      console.error('Forgot password error:', err);
      setAlertMsg('Could not send the reset link');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950 px-6">
      <div className="w-126 max-w-sm rounded-xl border border-zinc-800 bg-zinc-900 p-8 shadow-sm">
        <div className="mb-6 flex items-center justify-center gap-2 text-xl font-semibold text-zinc-100">
          QuitLARP
        </div>
        <h1 className="text-2xl font-semibold text-zinc-100">Reset your password</h1>
        <p className="mt-1 text-sm text-zinc-400">
          Enter your email and we'll send you a reset link
        </p>

        {alertMsg && (
          <div className="mt-4">
            <Alert kind="error" text={alertMsg} />
          </div>
        )}

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <label className="block">
            <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-400">
              Email
            </span>
            <input
              type="email"
              required
              autoComplete="email"
              value={resetEmail}
              onChange={(e) => setResetEmail(e.target.value)}
              placeholder="you@example.com"
              className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
            />
          </label>
          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-md bg-teal-500 px-4 py-2 text-sm font-medium text-zinc-950 hover:bg-teal-400 disabled:opacity-60"
          >
            {loading ? 'Sending…' : 'Send reset link'}
          </button>
          <Link
            to="/login"
            className="block w-full text-center text-sm text-zinc-400 hover:text-zinc-100"
          >
            ← Back to sign in
          </Link>
        </form>
      </div>
    </div>
  );
}

export default App;

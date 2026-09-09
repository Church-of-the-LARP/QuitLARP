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
    <div className="mx-auto w-full max-w-sm">
      <h1 className="text-2xl font-semibold">Reset your password</h1>
      <p className="mt-1 text-sm text-gray-500">
        Enter your email and we'll send you a reset link
      </p>

      {alertMsg && (
        <div className="mt-4">
          <Alert kind="error" text={alertMsg} />
        </div>
      )}

      <form onSubmit={handleSubmit} className="mt-6 space-y-4">
        <label className="block">
          <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500">
            Email
          </span>
          <input
            type="email"
            required
            autoComplete="email"
            value={resetEmail}
            onChange={(e) => setResetEmail(e.target.value)}
            placeholder="you@example.com"
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
          />
        </label>
        <button
          type="submit"
          disabled={loading}
          className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
        >
          {loading ? 'Sending…' : 'Send reset link'}
        </button>
        <Link
          to="/login"
          className="block w-full text-center text-sm text-gray-500 hover:text-gray-800"
        >
          ← Back to sign in
        </Link>
      </form>
    </div>
  );
}

export default App;

import { useEffect, useState, type FormEvent } from 'react';
import Alert from '../components/Alert.tsx';
import { Link, useLocation, useSearchParams } from 'react-router-dom';
import useAuth from '../scripts/useAuth.tsx';
import { apiUrl } from '../scripts/api.ts';
import type { ActionResult } from '../types/types.ts';

function App() {
  const googleUrl = `${apiUrl}/api/v1/auth/google`;
  const { loginUser } = useAuth();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const [loginEmail, setLoginEmail] = useState('');
  const [loginPass, setLoginPass] = useState('');
  const [loading, setLoading] = useState(false);
  const [alertMsg, setAlertMsg] = useState<ActionResult>(() => {
    const state = location.state as { notice?: string } | null;
    return state?.notice ? { notice: state.notice } : {};
  });
  // google bounces failed logins back here as ?error=...
  const [googleMsg] = useState(() => searchParams.get('error'));

  useEffect(() => {
    if (!searchParams.has('error')) return;

    const next = new URLSearchParams(searchParams);
    next.delete('error');
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams]);

  const handleLogin = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setAlertMsg({});
    try {
      setAlertMsg(await loginUser(loginEmail, loginPass));
    } catch (err) {
      console.error('Login error:', err);
      setAlertMsg({ error: 'Something went wrong while signing in' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950 px-6">
      <div className="w-126 rounded-xl border border-zinc-800 bg-zinc-900 p-8 shadow-sm">
        <div className="mb-6 flex items-center justify-center gap-2 text-xl font-semibold text-zinc-100">
          QuitLARP
        </div>
        <div className="mx-auto w-full max-w-sm">
          <h1 className="text-2xl font-semibold text-zinc-100">Sign in</h1>
          <p className="mt-1 text-sm text-zinc-400">to QuitLARP</p>

          {googleMsg && (
            <div className="mt-4">
              <Alert kind="error" text={googleMsg} />
            </div>
          )}
          {alertMsg.error && (
            <div className="mt-4">
              <Alert kind="error" text={alertMsg.error} />
            </div>
          )}
          {alertMsg.notice && (
            <div className="mt-4">
              <Alert kind="notice" text={alertMsg.notice} />
            </div>
          )}

          <form onSubmit={handleLogin} className="mt-6 space-y-4">
            <label className="block">
              <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-400">
                Email
              </span>
              <input
                type="email"
                required
                autoComplete="email"
                value={loginEmail}
                onChange={(e) => setLoginEmail(e.target.value)}
                placeholder="you@example.com"
                className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
              />
            </label>
            <label className="block">
              <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-400">
                Password
              </span>
              <input
                type="password"
                required
                autoComplete="current-password"
                value={loginPass}
                onChange={(e) => setLoginPass(e.target.value)}
                className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
              />
            </label>
            <button
              type="submit"
              disabled={loading}
              className="w-full rounded-md bg-teal-500 px-4 py-2 text-sm font-medium text-zinc-950 hover:bg-teal-400 disabled:opacity-60"
            >
              {loading ? 'Signing in…' : 'Sign in'}
            </button>
          </form>

          <div className="mt-3 text-right">
            <Link
              to="/forgot-password"
              className="text-sm text-zinc-400 hover:text-zinc-100"
            >
              Forgot password?
            </Link>
          </div>

          <div className="my-5 flex items-center gap-3 text-xs text-zinc-500">
            <span className="h-px flex-1 bg-zinc-800" />
            or
            <span className="h-px flex-1 bg-zinc-800" />
          </div>

          <a
            href={googleUrl}
            className="block w-full rounded-md border border-zinc-700 bg-zinc-800 px-4 py-2 text-center text-sm font-medium text-zinc-300 hover:bg-zinc-700"
          >
            Continue with Google
          </a>

          <p className="mt-5 text-center text-sm text-zinc-400">
            No account yet?{' '}
            <Link
              to="/register"
              className="font-medium text-teal-400 hover:underline"
            >
              Register
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}

export default App;

import { useEffect, useState } from 'react';
import { useForm } from '@tanstack/react-form';
import { z } from 'zod';
import Alert from '../components/Alert.tsx';
import TextField from '../components/TextField.tsx';
import { Link, useLocation, useSearchParams } from 'react-router-dom';
import useAuth from '../scripts/useAuth.tsx';
import { apiUrl } from '../scripts/api.ts';
import type { ActionResult } from '../types/types.ts';
import usePageTitle from '../scripts/usePageTitle.ts';
import Logo from '../components/Logo.tsx';

const schema = z.object({
  email: z.email('Enter a valid email'),
  password: z.string().min(1, 'Password is required'),
});

function App() {
  usePageTitle('Log in');
  const googleUrl = `${apiUrl}/api/v1/auth/google`;
  const { loginUser } = useAuth();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
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

  const form = useForm({
    defaultValues: {
      email: '',
      password: '',
    },
    validators: {
      onChange: schema,
    },
    onSubmit: async ({ value }) => {
      setAlertMsg({});
      try {
        setAlertMsg(await loginUser(value.email, value.password));
      } catch (err) {
        console.error('Login error:', err);
        setAlertMsg({ error: 'Something went wrong while signing in' });
      }
    },
  });

  return (
    <div className="flex min-h-screen items-center justify-center bg-shell px-6">
      <div className="w-126 rounded-xl border border-line bg-panel p-8 shadow-sm">
        <div className="mb-6 flex items-center justify-center gap-2">
          <Logo className="h-6 w-auto" />
        </div>
        <div className="mx-auto w-full max-w-sm">
          <h1 className="text-2xl font-semibold text-ink">Sign in</h1>
          <p className="mt-1 text-sm text-muted">to QuitLARP</p>

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

          <form
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              form.handleSubmit();
            }}
            className="mt-6 space-y-4"
          >
            <form.Field name="email">
              {(field) => (
                <TextField
                  field={field}
                  label="Email"
                  type="email"
                  autoComplete="email"
                  placeholder="you@example.com"
                />
              )}
            </form.Field>
            <form.Field name="password">
              {(field) => (
                <TextField
                  field={field}
                  label="Password"
                  type="password"
                  autoComplete="current-password"
                />
              )}
            </form.Field>
            <form.Subscribe selector={(s) => [s.canSubmit, s.isSubmitting]}>
              {([canSubmit, isSubmitting]) => (
                <button
                  type="submit"
                  disabled={!canSubmit}
                  className="w-full rounded-md bg-accent px-4 py-2 text-sm font-medium text-shell hover:bg-accent-light disabled:opacity-60"
                >
                  {isSubmitting ? 'Signing in…' : 'Sign in'}
                </button>
              )}
            </form.Subscribe>
          </form>

          <div className="mt-3 text-right">
            <Link
              to="/forgot-password"
              className="text-sm text-muted hover:text-ink"
            >
              Forgot password?
            </Link>
          </div>

          <div className="my-5 flex items-center gap-3 text-xs text-dim">
            <span className="h-px flex-1 bg-surface" />
            or
            <span className="h-px flex-1 bg-surface" />
          </div>

          <a
            href={googleUrl}
            className="block w-full rounded-md border border-line-hover bg-surface px-4 py-2 text-center text-sm font-medium text-muted hover:bg-raised"
          >
            Continue with Google
          </a>

          <p className="mt-5 text-center text-sm text-muted">
            No account yet?{' '}
            <Link
              to="/register"
              className="font-medium text-accent-light hover:underline"
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

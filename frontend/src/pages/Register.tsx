import { useState } from 'react';
import { useForm } from '@tanstack/react-form';
import { z } from 'zod';
import Alert from '../components/Alert.tsx';
import TextField from '../components/TextField.tsx';
import { Link } from 'react-router-dom';
import useAuth from '../scripts/useAuth.tsx';
import { apiUrl } from '../scripts/api.ts';
import type { ActionResult } from '../types/types.ts';

const schema = z.object({
  username: z
    .string()
    .min(3, 'At least 3 characters')
    .max(32, 'At most 32 characters')
    .regex(/^[A-Za-z0-9_-]+$/, 'Letters, numbers, - and _ only'),
  email: z.email('Enter a valid email'),
  password: z.string().min(8, 'At least 8 characters'),
});

function App() {
  const googleUrl = `${apiUrl}/api/v1/auth/google`;
  const { registerUser } = useAuth();
  const [alertMsg, setAlertMsg] = useState<ActionResult>({});

  const form = useForm({
    defaultValues: {
      username: '',
      email: '',
      password: '',
    },
    validators: {
      onChange: schema,
    },
    onSubmit: async ({ value }) => {
      setAlertMsg({});
      try {
        setAlertMsg(
          await registerUser(
            value.username.trim(),
            value.email.trim(),
            value.password,
          ),
        );
      } catch (err) {
        console.error('Register error:', err);
        setAlertMsg({ error: 'Something went wrong while creating the account' });
      }
    },
  });

  return (
    <div className="flex min-h-screen items-center justify-center bg-shell px-6">
      <div className="w-126 rounded-xl border border-line bg-panel p-8 shadow-sm">
        <div className="mb-6 flex items-center justify-center gap-2 text-xl font-semibold text-ink">
          QuitLARP
        </div>
        <div className="mx-auto w-full max-w-sm">
          <h1 className="text-2xl font-semibold text-ink">Create your account</h1>
          <p className="mt-1 text-sm text-muted">
            Username, email and a strong password
          </p>

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
            <form.Field name="username">
              {(field) => (
                <TextField
                  field={field}
                  label="Username"
                  autoComplete="username"
                  placeholder="jane_doe"
                />
              )}
            </form.Field>
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
            <p className="text-xs text-dim">At least 8 characters.</p>
            <form.Subscribe selector={(s) => [s.canSubmit, s.isSubmitting]}>
              {([canSubmit, isSubmitting]) => (
                <button
                  type="submit"
                  disabled={!canSubmit}
                  className="w-full rounded-md bg-accent px-4 py-2 text-sm font-medium text-shell hover:bg-accent-light disabled:opacity-60"
                >
                  {isSubmitting ? 'Creating account…' : 'Register'}
                </button>
              )}
            </form.Subscribe>
          </form>

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
            Already registered?{' '}
            <Link
              to="/login"
              className="font-medium text-accent-light hover:underline"
            >
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}

export default App;

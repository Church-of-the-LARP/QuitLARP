import { useState } from 'react';
import { useForm } from '@tanstack/react-form';
import { z } from 'zod';
import Alert from '../components/Alert.tsx';
import TextField from '../components/TextField.tsx';
import { Link, useNavigate } from 'react-router-dom';
import client, { getErrorText } from '../scripts/api.ts';
import usePageTitle from '../scripts/usePageTitle.ts';

const schema = z.object({
  email: z.email('Enter a valid email'),
});

function App() {
  usePageTitle('Forgot password');
  const navigate = useNavigate();
  const [alertMsg, setAlertMsg] = useState<string | null>(null);

  const form = useForm({
    defaultValues: {
      email: '',
    },
    validators: {
      onChange: schema,
    },
    onSubmit: async ({ value }) => {
      setAlertMsg(null);
      try {
        const res = await client.POST('/api/v1/auth/forgot-password', {
          body: { email: value.email.trim() },
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
      }
    },
  });

  return (
    <div className="flex min-h-screen items-center justify-center bg-shell px-6">
      <div className="w-126 max-w-sm rounded-xl border border-line bg-panel p-8 shadow-sm">
        <div className="mb-6 flex items-center justify-center gap-2 text-xl font-semibold text-ink">
          QuitLARP
        </div>
        <h1 className="text-2xl font-semibold text-ink">Reset your password</h1>
        <p className="mt-1 text-sm text-muted">
          Enter your email and we'll send you a reset link
        </p>

        {alertMsg && (
          <div className="mt-4">
            <Alert kind="error" text={alertMsg} />
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
          <form.Subscribe selector={(s) => [s.canSubmit, s.isSubmitting]}>
            {([canSubmit, isSubmitting]) => (
              <button
                type="submit"
                disabled={!canSubmit}
                className="w-full rounded-md bg-accent px-4 py-2 text-sm font-medium text-shell hover:bg-accent-light disabled:opacity-60"
              >
                {isSubmitting ? 'Sending…' : 'Send reset link'}
              </button>
            )}
          </form.Subscribe>
          <Link
            to="/login"
            className="block w-full text-center text-sm text-muted hover:text-ink"
          >
            ← Back to sign in
          </Link>
        </form>
      </div>
    </div>
  );
}

export default App;

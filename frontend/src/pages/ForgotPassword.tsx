import { useState } from 'react';
import { useForm } from '@tanstack/react-form';
import { z } from 'zod';
import Alert from '../components/Alert.tsx';
import TextField from '../components/TextField.tsx';
import { Link, useNavigate } from 'react-router-dom';
import client, { getErrorText } from '../scripts/api.ts';

const schema = z.object({
  email: z.email('Enter a valid email'),
});

function App() {
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
                className="w-full rounded-md bg-teal-500 px-4 py-2 text-sm font-medium text-zinc-950 hover:bg-teal-400 disabled:opacity-60"
              >
                {isSubmitting ? 'Sending…' : 'Send reset link'}
              </button>
            )}
          </form.Subscribe>
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

import { useState } from 'react';
import { useForm } from '@tanstack/react-form';
import { z } from 'zod';
import Alert from '../components/Alert.tsx';
import TextField from '../components/TextField.tsx';
import { Link, useSearchParams } from 'react-router-dom';
import client, { getErrorText } from '../scripts/api.ts';
import type { AlertMessage } from '../types/types.ts';

const schema = z
  .object({
    newPass: z.string().min(8, 'Password must be at least 8 characters'),
    confirmPass: z.string(),
  })
  .refine((v) => v.newPass === v.confirmPass, {
    message: 'Passwords do not match',
    path: ['confirmPass'],
  });

function App() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') ?? '';
  const [alertMsg, setAlertMsg] = useState<AlertMessage | null>(null);

  const form = useForm({
    defaultValues: {
      newPass: '',
      confirmPass: '',
    },
    validators: {
      onChange: schema,
    },
    onSubmit: async ({ value }) => {
      setAlertMsg(null);
      try {
        const res = await client.POST('/api/v1/auth/reset-password', {
          body: { token, newPassword: value.newPass },
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
            className="mt-6 block w-full rounded-md bg-teal-500 px-4 py-2 text-center text-sm font-medium text-zinc-950 hover:bg-teal-400"
          >
            Go to sign in
          </Link>
        ) : (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              e.stopPropagation();
              form.handleSubmit();
            }}
            className="mt-6 space-y-4"
          >
            <form.Field name="newPass">
              {(field) => (
                <TextField
                  field={field}
                  label="New password"
                  type="password"
                  autoComplete="new-password"
                />
              )}
            </form.Field>
            <form.Field name="confirmPass">
              {(field) => (
                <TextField
                  field={field}
                  label="Repeat new password"
                  type="password"
                  autoComplete="new-password"
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
                  {isSubmitting ? 'Updating…' : 'Update password'}
                </button>
              )}
            </form.Subscribe>
          </form>
        )}
      </div>
    </div>
  );
}

export default App;

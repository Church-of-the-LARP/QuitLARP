import { useState, type FormEvent } from "react";
import { useAuth, type ActionResult } from "../auth/AuthContext";
import { Field, Notice } from "../components/ui";

export function RegisterPage({ onShowLogin }: { onShowLogin: () => void }) {
  const { signUp, googleHref } = useAuth();
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [result, setResult] = useState<ActionResult>({});
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setResult({});
    try {
      setResult(await signUp(username.trim(), email.trim(), password));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto w-full max-w-sm">
      <h1 className="text-2xl font-semibold">Create your account</h1>
      <p className="mt-1 text-sm text-gray-500">Username, email and a strong password</p>

      {result.error && <div className="mt-4"><Notice kind="error">{result.error}</Notice></div>}
      {result.notice && <div className="mt-4"><Notice kind="notice">{result.notice}</Notice></div>}

      <form onSubmit={submit} className="mt-6 space-y-4">
        <Field
          label="Username"
          required
          minLength={3}
          maxLength={32}
          pattern="[A-Za-z0-9_-]+"
          autoComplete="username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="jane_doe"
        />
        <Field
          label="Email"
          type="email"
          required
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
        />
        <Field
          label="Password"
          type="password"
          required
          minLength={8}
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
        <p className="text-xs text-gray-400">At least 8 characters.</p>
        <button
          type="submit"
          disabled={busy}
          className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
        >
          {busy ? "Creating account…" : "Register"}
        </button>
      </form>

      <div className="my-5 flex items-center gap-3 text-xs text-gray-400">
        <span className="h-px flex-1 bg-gray-200" />
        or
        <span className="h-px flex-1 bg-gray-200" />
      </div>

      <a
        href={googleHref}
        className="block w-full rounded-md border border-gray-300 bg-white px-4 py-2 text-center text-sm font-medium text-gray-700 hover:bg-gray-50"
      >
        Continue with Google
      </a>

      <p className="mt-5 text-center text-sm text-gray-500">
        Already registered?{" "}
        <button
          type="button"
          onClick={onShowLogin}
          className="font-medium text-blue-600 hover:underline"
        >
          Sign in
        </button>
      </p>
    </div>
  );
}

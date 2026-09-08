import { useState, type FormEvent } from "react";
import { useAuth, type ActionResult } from "../auth/AuthContext";
import { client, errorMessage } from "../api/client";
import { Field, Notice } from "../components/ui";

export function LoginPage({
  onShowRegister,
  googleNotice,
}: {
  onShowRegister: () => void;
  googleNotice?: string;
}) {
  const { signIn, googleHref } = useAuth();
  const [mode, setMode] = useState<"login" | "forgot">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [result, setResult] = useState<ActionResult>({});
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setResult({});
    try {
      setResult(await signIn(email, password));
    } finally {
      setBusy(false);
    }
  }

  async function requestReset(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setResult({});
    try {
      const res = await client.POST("/api/v1/auth/forgot-password", {
        body: { email: email.trim() },
      });
      setResult(
        res.error
          ? { error: errorMessage(res.error, "Could not send the reset link") }
          : {
              notice:
                res.data?.message ??
                "If an account exists for that email, a reset link is on its way.",
            },
      );
      setMode("login");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto w-full max-w-sm">
      <h1 className="text-2xl font-semibold">Sign in</h1>
      <p className="mt-1 text-sm text-gray-500">to QuitLARP</p>

      {googleNotice && <div className="mt-4"><Notice kind="error">{googleNotice}</Notice></div>}
      {result.error && <div className="mt-4"><Notice kind="error">{result.error}</Notice></div>}
      {result.notice && <div className="mt-4"><Notice kind="notice">{result.notice}</Notice></div>}

      {mode === "forgot" ? (
        <form onSubmit={requestReset} className="mt-6 space-y-4">
          <Field
            label="Email"
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
          />
          <button
            type="submit"
            disabled={busy}
            className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
          >
            {busy ? "Sending…" : "Send reset link"}
          </button>
          <button
            type="button"
            onClick={() => setMode("login")}
            className="w-full text-center text-sm text-gray-500 hover:text-gray-800"
          >
            ← Back to sign in
          </button>
        </form>
      ) : (
        <>
          <form onSubmit={submit} className="mt-6 space-y-4">
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
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            <button
              type="submit"
              disabled={busy}
              className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
            >
              {busy ? "Signing in…" : "Sign in"}
            </button>
          </form>

          <div className="mt-3 text-right">
            <button
              type="button"
              onClick={() => setMode("forgot")}
              className="text-sm text-gray-500 hover:text-gray-800"
            >
              Forgot password?
            </button>
          </div>

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
            No account yet?{" "}
            <button
              type="button"
              onClick={onShowRegister}
              className="font-medium text-blue-600 hover:underline"
            >
              Register
            </button>
          </p>
        </>
      )}
    </div>
  );
}

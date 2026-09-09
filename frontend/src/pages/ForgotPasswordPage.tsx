import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router-dom";
import { client, errorMessage } from "../api/client";
import { Notice } from "../components/Notice";

export function ForgotPasswordPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await client.POST("/api/v1/auth/forgot-password", {
        body: { email: email.trim() },
      });
      if (res.error) {
        setError(errorMessage(res.error, "Could not send the reset link"));
        return;
      }
      navigate("/login", {
        state: {
          notice:
            res.data?.message ??
            "If an account exists for that email, a reset link is on its way.",
        },
      });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto w-full max-w-sm">
      <h1 className="text-2xl font-semibold">Reset your password</h1>
      <p className="mt-1 text-sm text-gray-500">
        Enter your email and we'll send you a reset link
      </p>

      {error && (
        <div className="mt-4">
          <Notice kind="error">{error}</Notice>
        </div>
      )}

      <form onSubmit={submit} className="mt-6 space-y-4">
        <label className="block">
          <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500">
            Email
          </span>
          <input
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
          />
        </label>
        <button
          type="submit"
          disabled={busy}
          className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
        >
          {busy ? "Sending…" : "Send reset link"}
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

import { useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { client, errorMessage } from "../api/client";
import { Notice } from "../components/Notice";

export function ResetPasswordPage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [result, setResult] = useState<{
    kind: "error" | "notice";
    text: string;
  } | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setResult(null);
    if (password.length < 8) {
      setResult({
        kind: "error",
        text: "Password must be at least 8 characters",
      });
      return;
    }
    if (password !== confirm) {
      setResult({ kind: "error", text: "Passwords do not match" });
      return;
    }
    setBusy(true);
    try {
      const res = await client.POST("/api/v1/auth/reset-password", {
        body: { token, newPassword: password },
      });
      setResult(
        res.error
          ? {
              kind: "error",
              text: errorMessage(res.error, "Could not reset the password"),
            }
          : { kind: "notice", text: res.data?.message ?? "Password updated." },
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto w-full max-w-sm">
      <h1 className="text-2xl font-semibold">Reset your password</h1>
      <p className="mt-1 text-sm text-gray-500">
        Choose a new password for your account
      </p>

      {result && (
        <div className="mt-4">
          <Notice kind={result.kind}>{result.text}</Notice>
        </div>
      )}

      {result?.kind === "notice" ? (
        <Link
          to="/login"
          className="mt-6 block w-full rounded-md bg-blue-600 px-4 py-2 text-center text-sm font-medium text-white hover:bg-blue-700"
        >
          Go to sign in
        </Link>
      ) : (
        <form onSubmit={submit} className="mt-6 space-y-4">
          <label className="block">
            <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500">
              New password
            </span>
            <input
              type="password"
              required
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-gray-500">
              Repeat new password
            </span>
            <input
              type="password"
              required
              autoComplete="new-password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
          </label>
          <button
            type="submit"
            disabled={busy}
            className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
          >
            {busy ? "Updating…" : "Update password"}
          </button>
        </form>
      )}
    </div>
  );
}

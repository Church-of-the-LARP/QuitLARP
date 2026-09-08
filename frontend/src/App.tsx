import { useEffect, useState, type ReactNode } from "react";
import { AuthProvider, useAuth, type ActionResult } from "./auth/AuthContext";
import { Notice, RoleBadge } from "./components/ui";
import { LoginPage } from "./pages/LoginPage";
import { RegisterPage } from "./pages/RegisterPage";
import { ResetPasswordPage } from "./pages/ResetPasswordPage";
import { UsersPage, canViewUsers } from "./pages/UsersPage";

/**
 * Top-level shell: restores the session, decides between the auth gate and
 * the signed-in area, and handles email-link routes (/reset-password) and
 * the ?error= notice that the OAuth callback redirects with.
 */
function Shell() {
  const { status, user, signOut, resendVerification } = useAuth();
  const [showRegister, setShowRegister] = useState(false);
  const [googleError, setGoogleError] = useState<string | null>(null);
  const [verifyResult, setVerifyResult] = useState<ActionResult>({});

  // Reset flow: /reset-password?token=...
  const params = new URLSearchParams(window.location.search);
  const resetToken =
    window.location.pathname === "/reset-password" ? (params.get("token") ?? "") : "";
  const onResetFlow = resetToken.length > 0;

  // OAuth callback errors arrive as ?error=... on the frontend URL.
  useEffect(() => {
    const error = new URLSearchParams(window.location.search).get("error");
    if (error) {
      setGoogleError(error);
      window.history.replaceState({}, "", window.location.pathname);
    }
  }, []);

  if (status === "loading") {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-gray-400">
        Loading…
      </div>
    );
  }

  if (onResetFlow) {
    return (
      <AuthGate>
        <ResetPasswordPage token={resetToken} />
      </AuthGate>
    );
  }

  if (!user) {
    return (
      <AuthGate>
        {showRegister ? (
          <RegisterPage onShowLogin={() => setShowRegister(false)} />
        ) : (
          <LoginPage
            onShowRegister={() => setShowRegister(true)}
            googleNotice={googleError ?? undefined}
          />
        )}
      </AuthGate>
    );
  }

  async function resend() {
    setVerifyResult(await resendVerification());
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-6 py-3">
          <p className="text-sm font-semibold text-gray-900">QuitLARP</p>
          <div className="flex items-center gap-3">
            <span className="hidden text-sm text-gray-600 sm:inline">
              {user.username}
              <span className="mx-1.5 text-gray-300">·</span>
              {user.email}
            </span>
            <RoleBadge role={user.role} />
            <button
              onClick={() => void signOut()}
              className="rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100"
            >
              Sign out
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-6 py-8">
        {!user.emailVerified && (
          <div className="mb-6 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
            <p className="font-medium">Verify your email address</p>
            <p className="mt-0.5 text-amber-700">
              We emailed a verification link to <span className="font-semibold">{user.email}</span>.
              {" "}Didn't get it?{" "}
              <button
                onClick={() => void resend()}
                className="font-semibold underline underline-offset-2 hover:text-amber-900"
              >
                Resend the email
              </button>
              .
            </p>
            {verifyResult.error && <Notice kind="error">{verifyResult.error}</Notice>}
            {verifyResult.notice && <Notice kind="notice">{verifyResult.notice}</Notice>}
          </div>
        )}

        <div className="mb-6 rounded-md border border-gray-200 bg-white px-5 py-4">
          <p className="text-xs font-medium uppercase tracking-wide text-gray-400">
            Signed in as
          </p>
          <p className="mt-1 text-sm text-gray-800">
            <span className="font-semibold">{user.username}</span>
            {user.googleLinked && (
              <span className="ml-2 rounded bg-blue-50 px-1.5 py-0.5 text-xs text-blue-700">
                google account
              </span>
            )}
          </p>
        </div>

        {canViewUsers(user.role) && <UsersPage />}
      </main>
    </div>
  );
}

function AuthGate({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-6">
      <div className="w-full rounded-xl border border-gray-200 bg-white p-8 shadow-sm">
        <div className="mb-6 flex items-center justify-center gap-2 text-xl font-semibold text-gray-900">
          QuitLARP
        </div>
        {children}
      </div>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <Shell />
    </AuthProvider>
  );
}

import { Link, Outlet } from "react-router-dom";
import useAuth from "../scripts/useAuth";
import type { Role } from "../types/types";

export default function Layout() {
  const { user, logoutUser } = useAuth();

  const handleLogout = async () => {
    await logoutUser();
  };

  if (!user) return null;

  const isAdmin = user.role === "admin" || user.role === "superadmin";

  const roleStyles: Record<Role, string> = {
    user: "bg-zinc-700 text-zinc-300",
    admin: "bg-teal-400/20 text-teal-400",
    superadmin: "bg-amber-400/20 text-amber-400",
  };

  return (
    <>
      <div className="min-h-screen bg-zinc-950">
        <header className="border-b border-zinc-800 bg-zinc-900">
          <div className="mx-auto flex w-[75vw] items-center justify-between py-3">
            <div className="flex items-center gap-4">
              <Link to="/" className="text-sm font-semibold text-zinc-100">
                QuitLARP
              </Link>
              {isAdmin && (
                <Link
                  to="/users"
                  className="text-sm text-zinc-400 hover:text-zinc-100"
                >
                  Users
                </Link>
              )}
            </div>
            <div className="flex items-center gap-3">
              <span className="hidden text-sm text-zinc-400 sm:inline">
                {user.username}
                <span className="mx-1.5 text-zinc-600">·</span>
                {user.email}
              </span>
              <span
                className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[user.role]}`}
              >
                {user.role}
              </span>
              <button
                onClick={() => handleLogout()}
                className="rounded-md border border-zinc-700 px-3 py-1.5 text-xs font-medium text-zinc-400 hover:bg-zinc-800"
              >
                Sign out
              </button>
            </div>
          </div>
        </header>
        <Outlet />
      </div>
    </>
  );
}

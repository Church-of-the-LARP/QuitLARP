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
    user: "bg-gray-100 text-gray-700",
    admin: "bg-violet-100 text-violet-700",
    superadmin: "bg-amber-100 text-amber-800",
  };

  return (
    <>
      <div className="min-h-screen bg-gray-50">
        <header className="border-b border-gray-200 bg-white">
          <div className="mx-auto flex w-[75vw] items-center justify-between py-3">
            <div className="flex items-center gap-4">
              <Link to="/" className="text-sm font-semibold text-gray-900">
                QuitLARP
              </Link>
              {isAdmin && (
                <Link
                  to="/users"
                  className="text-sm text-gray-500 hover:text-gray-800"
                >
                  Users
                </Link>
              )}
            </div>
            <div className="flex items-center gap-3">
              <span className="hidden text-sm text-gray-600 sm:inline">
                {user.username}
                <span className="mx-1.5 text-gray-300">·</span>
                {user.email}
              </span>
              <span
                className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[user.role]}`}
              >
                {user.role}
              </span>
              <button
                onClick={() => handleLogout()}
                className="rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-100"
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

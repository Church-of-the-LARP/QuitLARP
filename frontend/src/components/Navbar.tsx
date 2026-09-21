import { Link } from "react-router-dom";

type NavbarProps = {
  user: {
    username: string;
    email: string;
    role: "user" | "admin" | "superadmin";
  };
  isAdmin: boolean;
  roleStyles: Record<"user" | "admin" | "superadmin", string>;
  handleLogout: () => void;
}

export default function Navbar({ user, isAdmin, roleStyles, handleLogout }: NavbarProps) {
  return (
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
  );
}

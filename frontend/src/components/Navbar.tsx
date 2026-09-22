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
    <header className="border-b border-line bg-panel">
      <div className="mx-auto flex w-[75vw] items-center justify-between py-3">
        <div className="flex items-center gap-4">
          <Link to="/" className="text-sm font-semibold text-ink">
            QuitLARP
          </Link>
          {isAdmin && (
            <Link
              to="/users"
              className="text-sm text-muted hover:text-ink"
            >
              Users
            </Link>
          )}
        </div>
        <div className="flex items-center gap-6">
          <span className="hidden text-sm text-muted sm:inline">
            {user.username}
            <span className="mx-1.5 text-dim">·</span>
            {user.email}
          </span>
          <span
            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${roleStyles[user.role]}`}
          >
            {user.role}
          </span>
          <button
            onClick={() => handleLogout()}
            className="text-sm font-semibold text-ink py-2 px-4 rounded-2xl bg-surface hover:bg-raised transition-colors"
          >
            Sign out
          </button>
        </div>
      </div>
    </header>
  );
}

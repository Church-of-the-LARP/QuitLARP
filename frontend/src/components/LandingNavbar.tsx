import { Link } from "react-router-dom";

export default function LandingNavbar() {
  return (
    <header className="relative z-20 border-b border-line bg-panel shadow-lg shadow-black/30">
      <div className="mx-auto flex w-[75vw] items-center justify-between py-3">
        <Link to="/" className="text-sm font-semibold text-ink">
          QuitLARP
        </Link>
        <div className="flex items-center gap-6">
          <Link
            to="/login"
            className="text-sm font-semibold text-ink py-2 px-4 rounded-2xl hover:bg-surface transition-colors"
          >
            Login
          </Link>
          <Link
            to="/register"
            className="text-sm font-semibold text-ink py-2 px-4 rounded-2xl bg-surface hover:bg-raised transition-colors"
          >
            Register
          </Link>
        </div>
      </div>
    </header>
  );
}

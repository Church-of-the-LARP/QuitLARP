import { Link } from "react-router-dom";

export default function LandingNavbar() {
  return (
    <header className="border-b border-zinc-800 bg-zinc-900">
      <div className="mx-auto flex w-[75vw] items-center justify-between py-3">
        <Link to="/" className="text-sm font-semibold text-zinc-100">
          QuitLARP
        </Link>
        <div className="flex items-center gap-6">
          <Link to="/login" className="text-sm font-semibold text-zinc-100">
            Login
          </Link>
          <Link to="/register" className="text-sm font-semibold text-zinc-100">
            Register
          </Link>
        </div>
      </div>
    </header>
  );
}

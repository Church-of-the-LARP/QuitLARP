import { Outlet } from "react-router-dom";
import useAuth from "../scripts/useAuth";
import type { Role } from "../types/types";
import Navbar from "./Navbar";

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
        <Navbar
          user={user}
          isAdmin={isAdmin}
          roleStyles={roleStyles}
          handleLogout={handleLogout}
        />
        <Outlet />
      </div>
    </>
  );
}

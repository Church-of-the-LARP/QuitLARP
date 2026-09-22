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
    user: "bg-raised text-muted",
    admin: "bg-accent-light/20 text-accent-light",
    superadmin: "bg-warning/20 text-warning",
  };

  return (
    <>
      <div className="min-h-screen bg-shell">
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

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

  const isAdmin = user.role === "admin";

  const roleStyles: Record<Role, string> = {
    user: "bg-raised text-muted",
    moderator: "bg-accent-secondary/20 text-accent-secondary",
    admin: "bg-warning/20 text-warning",
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

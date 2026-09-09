import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuth, type Role } from "../auth/AuthContext";
import { LoadingScreen } from "./LoadingScreen";

export function SecuredRoute({ roles }: { roles?: Role[] }) {
  const { status, user } = useAuth();
  const location = useLocation();

  if (status === "loading") {
    return <LoadingScreen />;
  }

  if (!user) {
    return (
      <Navigate to={{ pathname: "/login", search: location.search }} replace />
    );
  }

  if (roles && !roles.includes(user.role)) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}

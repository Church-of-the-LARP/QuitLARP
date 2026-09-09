import { Navigate, Outlet, useLocation } from 'react-router-dom';
import Loading from './Loading.tsx';
import useAuth from '../scripts/useAuth.tsx';
import type { Role } from '../types/types.ts';

export default function SecuredRoute({ roles }: { roles?: Role[] }) {
  const { status, user } = useAuth();
  const location = useLocation();

  if (status === 'loading') return <Loading />;

  if (!user) {
    return (
      <Navigate to={{ pathname: '/login', search: location.search }} replace />
    );
  }

  if (roles && !roles.includes(user.role)) return <Navigate to="/" replace />;

  return <Outlet />;
}

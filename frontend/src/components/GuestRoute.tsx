import { Navigate, Outlet } from 'react-router-dom';
import Loading from './Loading.tsx';
import useAuth from '../scripts/useAuth.tsx';

export default function GuestRoute() {
  const { status } = useAuth();

  if (status === 'loading') return <Loading />;
  if (status === 'authenticated') return <Navigate to="/" replace />;

  return <Outlet />;
}

import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore, type UserRole } from '../store/authStore';

interface Props {
  roles: UserRole[];
}

export function RoleRoute({ roles }: Props) {
  const user = useAuthStore((s) => s.user);
  if (!user) return <Navigate to="/login" replace />;
  if (!roles.includes(user.role)) return <Navigate to="/dashboard" replace />;
  return <Outlet />;
}

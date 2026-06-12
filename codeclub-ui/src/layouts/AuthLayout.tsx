import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';

export function AuthLayout() {
  const user = useAuthStore((s) => s.user);
  if (user) return <Navigate to="/dashboard" replace />;
  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
      <Outlet />
    </div>
  );
}

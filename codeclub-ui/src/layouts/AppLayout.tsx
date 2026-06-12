import { Outlet } from 'react-router-dom';
import { Sidebar } from '../components/layout/Sidebar';

export function AppLayout() {
  return (
    <div className="flex min-h-screen bg-gray-100">
      <Sidebar />
      <main className="flex-1 p-8 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}

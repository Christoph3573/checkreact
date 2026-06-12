import { NavLink } from 'react-router-dom';
import { useAuthStore, type UserRole } from '../../store/authStore';

interface NavItem {
  to: string;
  label: string;
  roles: UserRole[];
}

const navItems: NavItem[] = [
  { to: '/dashboard', label: 'Dashboard', roles: ['student', 'teacher', 'admin'] },
  { to: '/substitutions', label: 'Vertretungsplan', roles: ['student', 'teacher', 'admin'] },
  { to: '/calendar', label: 'Kalender', roles: ['student', 'teacher', 'admin'] },
  { to: '/homework', label: 'Hausaufgaben', roles: ['student', 'teacher', 'admin'] },
  { to: '/files', label: 'Dateien', roles: ['student', 'teacher', 'admin'] },
  { to: '/chat', label: 'Chat', roles: ['student', 'teacher', 'admin'] },
  { to: '/admin', label: 'Verwaltung', roles: ['admin'] },
];

export function Sidebar() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  const visible = navItems.filter((item) => user && item.roles.includes(user.role));

  return (
    <aside className="w-56 bg-white border-r border-gray-200 flex flex-col shrink-0">
      <div className="p-4 border-b border-gray-200">
        <span className="font-bold text-lg text-gray-900">Schul-App</span>
      </div>

      <nav className="flex-1 p-2 space-y-0.5">
        {visible.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              `block px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-blue-50 text-blue-700'
                  : 'text-gray-700 hover:bg-gray-100'
              }`
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>

      {user && (
        <div className="p-4 border-t border-gray-200">
          <p className="text-sm font-medium text-gray-900 truncate">
            {user.first_name} {user.last_name}
          </p>
          <p className="text-xs text-gray-500 truncate mb-2">{user.email}</p>
          <button
            onClick={logout}
            className="text-xs text-red-600 hover:text-red-800 transition-colors"
          >
            Abmelden
          </button>
        </div>
      )}
    </aside>
  );
}

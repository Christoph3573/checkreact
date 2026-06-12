import { useAuthStore } from '../../store/authStore';

const roleLabel: Record<string, string> = {
  student: 'Schüler-Dashboard',
  teacher: 'Lehrer-Dashboard',
  admin: 'Admin-Dashboard',
};

export function DashboardPage() {
  const user = useAuthStore((s) => s.user);

  return (
    <div>
      <h1 className="text-3xl font-bold text-gray-900 mb-1">
        Hallo, {user?.first_name}
      </h1>
      <p className="text-gray-500 mb-8 text-sm">
        {user ? roleLabel[user.role] : ''}
      </p>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
          <h2 className="font-semibold text-gray-700 mb-1 text-sm">Vertretungsplan</h2>
          <p className="text-sm text-gray-400">Keine Einträge für heute</p>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
          <h2 className="font-semibold text-gray-700 mb-1 text-sm">Hausaufgaben</h2>
          <p className="text-sm text-gray-400">Keine offenen Aufgaben</p>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
          <h2 className="font-semibold text-gray-700 mb-1 text-sm">Nächste Termine</h2>
          <p className="text-sm text-gray-400">Keine anstehenden Termine</p>
        </div>
      </div>
    </div>
  );
}

import { useCallback, useEffect, useState } from "react";

type Project = {
  id: number;
  slug: string;
  title: string;
  description?: string;
  created_at: string;
  updated_at: string;
};

type ProjectCardProps = {
  project: Project;
};

function ProjectCard({ project }: ProjectCardProps) {
  return (
    <div className="rounded-xl border bg-white p-4 shadow">
      <h2 className="text-xl font-bold">{project.title}</h2>

      <p className="text-gray-500">/{project.slug}</p>

      {project.description && (
        <p className="mt-2">{project.description}</p>
      )}
    </div>
  );
}

type ProjectListProps = {
  projects: Project[];
};

function ProjectList({ projects }: ProjectListProps) {
  return (
    <div className="grid gap-4">
      {projects.map((project) => (
        <ProjectCard key={project.id} project={project} />
      ))}
    </div>
  );
}

const apiBase = import.meta.env.VITE_API_BASE;
const user = import.meta.env.VITE_API_USER;
const password = import.meta.env.VITE_API_PASSWORD;

function basicAuthHeader(user: string, password: string): string {
  const token = btoa(`${user}:${password}`);
  return `Basic ${token}`;
}

const headers = {
  Accept: "application/json",
  "Content-Type": "application/json",
  Authorization: basicAuthHeader(user, password),
};

async function apiFetch<T>(path: string): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    headers,
  });

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }

  return response.json() as Promise<T>;
}

function App() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadProjects = useCallback(async () => {
    try {
      const data = await apiFetch<Project[]>("/api/v1/projects");
      setProjects(data);
      setError(null);
    } catch (err) {
      console.error(err);
      setError("Fehler beim Laden der Projekte.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProjects();

    const intervalId = setInterval(() => {
      loadProjects();
    }, 5000);

    return () => {
      clearInterval(intervalId);
    };
  }, [loadProjects]);

  return (
    <main className="min-h-screen bg-gray-100 p-8">
      <div className="mx-auto max-w-3xl">
        <h1 className="mb-2 text-4xl font-bold">
          Projekte
        </h1>

        <p className="mb-8 text-gray-600">
          Projekte aus der Codeclub-API
        </p>

        {loading && (
          <p className="mb-4 text-gray-500">
            Lade Projekte...
          </p>
        )}

        {error && (
          <p className="mb-4 text-red-600">
            {error}
          </p>
        )}

        {!loading && !error && (
          <ProjectList projects={projects} />
        )}
      </div>
    </main>
  );
}

export default App;
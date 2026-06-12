import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { useAuthStore } from '../store/authStore';

const apiBase = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080';

export const apiClient = axios.create({
  baseURL: `${apiBase}/api/v1`,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true, // send HttpOnly refresh-token cookie
});

// Attach access token to every request
apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// On 401: try to refresh once, then retry the original request
let isRefreshing = false;
let waitQueue: Array<{ resolve: (token: string) => void; reject: (err: unknown) => void }> = [];

function flushQueue(error: unknown, token: string | null) {
  waitQueue.forEach(({ resolve, reject }) => (error ? reject(error) : resolve(token!)));
  waitQueue = [];
}

apiClient.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const original = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    if (error.response?.status !== 401 || original._retry) {
      return Promise.reject(error);
    }

    if (isRefreshing) {
      return new Promise<string>((resolve, reject) => {
        waitQueue.push({ resolve, reject });
      }).then((token) => {
        original.headers.Authorization = `Bearer ${token}`;
        return apiClient(original);
      });
    }

    original._retry = true;
    isRefreshing = true;

    try {
      const { data } = await axios.post<{ access_token: string }>(
        `${apiBase}/api/v1/auth/refresh`,
        {},
        { withCredentials: true },
      );
      const newToken = data.access_token;
      useAuthStore.getState().setAccessToken(newToken);
      flushQueue(null, newToken);
      original.headers.Authorization = `Bearer ${newToken}`;
      return apiClient(original);
    } catch (err) {
      flushQueue(err, null);
      useAuthStore.getState().logout();
      return Promise.reject(err);
    } finally {
      isRefreshing = false;
    }
  },
);

import axios from 'axios';

const resolveApiUrl = () => {
  if (import.meta.env.VITE_API_URL) {
    return import.meta.env.VITE_API_URL;
  }
  if (typeof window !== 'undefined') {
    const { protocol, hostname, port } = window.location;
    const originPort = port ? `:${port}` : '';
    return `${protocol}//${hostname}${originPort}`;
  }
  return 'http://localhost:8081';
};

const API_URL = resolveApiUrl();

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error.response?.status;
    const requestUrl = error.config?.url || '';
    const authEndpoints = ['/api/auth/login', '/api/init/check', '/api/init/setup', '/api/auth/2fa/verify'];
    const shouldSkipRedirect = authEndpoints.some((endpoint) => requestUrl.includes(endpoint));

    if (status === 401 && !shouldSkipRedirect) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export const authAPI = {
  checkInit: () => api.get('/api/init/check'),
  initSetup: (data) => api.post('/api/init/setup', data),
  login: (credentials) => api.post('/api/auth/login', credentials),
};

export const usersAPI = {
  getAll: () => api.get('/api/users'),
  getById: (id) => api.get(`/api/users/${id}`),
  create: (data) => api.post('/api/users', data),
  update: (id, data) => api.put(`/api/users/${id}`, data),
  delete: (id) => api.delete(`/api/users/${id}`),
};

export const statsAPI = {
  getDashboard: () => api.get('/api/stats/dashboard'),
  getTraffic: (params = {}) => api.get('/api/stats/traffic', { params }),
};

export const logsAPI = {
  getRequests: (params = {}) => api.get('/api/logs/requests', { params }),
  exportRequests: (params = {}) =>
    api.get('/api/logs/requests/export', { params, responseType: 'blob' }),
  getRetention: () => api.get('/api/logs/retention'),
  clearLogs: (payload) => api.post('/api/logs/clear', payload),
};

export const auditAPI = {
  getLogs: (params = {}) => api.get('/api/audit/logs', { params }),
};

export const settingsAPI = {
  getAll: () => api.get('/api/settings'),
  update: (key, value) => api.put('/api/settings', { key, value }),
};

export const systemAPI = {
  getPublicIp: () => api.get('/api/system/public-ip'),
};

export const twoFactorAPI = {
  setup: () => api.post('/api/auth/2fa/setup'),
  verifySetup: (code) => api.post('/api/auth/2fa/verify-setup', { code }),
  disable: (payload) => api.post('/api/auth/2fa/disable', payload),
  regenerateCodes: (params) => api.get('/api/auth/2fa/backup-codes', { params }),
  verifyLogin: (payload, tempToken) =>
    api.post('/api/auth/2fa/verify', payload, {
      headers: tempToken
        ? {
            Authorization: `Bearer ${tempToken}`,
          }
        : {},
    }),
};

export default api;

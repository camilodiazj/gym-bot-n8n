/**
 * Application configuration loaded from environment variables.
 * Vite exposes env vars prefixed with VITE_ on import.meta.env
 */

const config = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  devUserId: import.meta.env.VITE_DEV_USER_ID || undefined,
};

export default config;

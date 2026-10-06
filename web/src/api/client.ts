import { AuthTokens, User } from '../types';

let memoryAccessToken: string | null = null;
let isRefreshing = false;
let refreshSubscribers: ((token: string) => void)[] = [];

export const getAccessToken = (): string | null => memoryAccessToken;
export const setAccessToken = (token: string | null) => {
  memoryAccessToken = token;
};

export const getRefreshToken = (): string | null => {
  return localStorage.getItem('planly_refresh_token');
};

export const setRefreshToken = (token: string | null) => {
  if (token) {
    localStorage.setItem('planly_refresh_token', token);
  } else {
    localStorage.removeItem('planly_refresh_token');
  }
};

const onRefreshed = (token: string) => {
  refreshSubscribers.forEach((callback) => callback(token));
  refreshSubscribers = [];
};

export interface ApiResponse<T> {
  data: T;
  totalCount?: number;
}

export class ApiError extends Error {
  code: string;
  status: number;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

export async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const url = endpoint.startsWith('http') ? endpoint : `/api/v1${endpoint.startsWith('/') ? '' : '/'}${endpoint}`;
  const headers = new Headers(options.headers || {});

  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  const token = getAccessToken();
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  let response: Response;
  try {
    response = await fetch(url, { ...options, headers });
  } catch (err: unknown) {
    const errorMsg = err instanceof Error ? err.message : 'Network request failed';
    throw new ApiError(0, 'network_error', errorMsg);
  }

  // Handle 401 Unauthorized with token refresh (avoid infinite loop on /auth endpoints)
  if (response.status === 401 && !endpoint.includes('/auth/login') && !endpoint.includes('/auth/refresh') && !endpoint.includes('/auth/register')) {
    const refreshToken = getRefreshToken();
    if (!refreshToken) {
      setAccessToken(null);
      window.dispatchEvent(new CustomEvent('auth:expired'));
      throw new ApiError(401, 'unauthorized', 'Session expired');
    }

    if (!isRefreshing) {
      isRefreshing = true;
      try {
        const refreshRes = await fetch('/api/v1/auth/refresh', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });

        if (!refreshRes.ok) {
          throw new Error('Refresh failed');
        }

        const data: AuthTokens = await refreshRes.json();
        setAccessToken(data.access_token);
        setRefreshToken(data.refresh_token);
        isRefreshing = false;
        onRefreshed(data.access_token);

        // Retry original request with new token
        headers.set('Authorization', `Bearer ${data.access_token}`);
        response = await fetch(url, { ...options, headers });
      } catch {
        isRefreshing = false;
        refreshSubscribers = [];
        setAccessToken(null);
        setRefreshToken(null);
        window.dispatchEvent(new CustomEvent('auth:expired'));
        throw new ApiError(401, 'unauthorized', 'Session expired. Please log in again.');
      }
    } else {
      // Wait for existing refresh to finish
      const retryToken = await new Promise<string>((resolve) => {
        refreshSubscribers.push(resolve);
      });
      headers.set('Authorization', `Bearer ${retryToken}`);
      response = await fetch(url, { ...options, headers });
    }
  }

  const totalCountHeader = response.headers.get('X-Total-Count');
  const totalCount = totalCountHeader ? parseInt(totalCountHeader, 10) : undefined;

  if (!response.ok) {
    let errorCode = 'unknown_error';
    let errorMessage = `Request failed with status ${response.status}`;
    try {
      const errJson = await response.json();
      if (errJson && errJson.error) {
        errorCode = errJson.error.code || errorCode;
        errorMessage = errJson.error.message || errorMessage;
      }
    } catch {
      // Use fallback error message
    }
    throw new ApiError(response.status, errorCode, errorMessage);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return { data: null as unknown as T, totalCount };
  }

  const data = await response.json();
  return { data, totalCount };
}

export const api = {
  get: <T>(endpoint: string, options?: RequestInit) =>
    request<T>(endpoint, { ...options, method: 'GET' }),
  post: <T>(endpoint: string, body?: unknown, options?: RequestInit) =>
    request<T>(endpoint, {
      ...options,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    }),
  put: <T>(endpoint: string, body?: unknown, options?: RequestInit) =>
    request<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    }),
  delete: <T>(endpoint: string, options?: RequestInit) =>
    request<T>(endpoint, { ...options, method: 'DELETE' }),
};

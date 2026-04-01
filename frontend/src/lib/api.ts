export type ApiError = { message: string }

async function request<T>(
  path: string,
  init: RequestInit & { json?: unknown } = {},
): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.json !== undefined) {
    headers.set('content-type', 'application/json')
  }
  const res = await fetch(path, {
    ...init,
    headers,
    credentials: 'include',
    body: init.json !== undefined ? JSON.stringify(init.json) : init.body,
  })
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`
    try {
      const j = (await res.json()) as ApiError
      if (j?.message) msg = j.message
    } catch {
      // ignore
    }
    throw new Error(msg)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export type User = { id: string; email: string; created_at: string; disabled_at?: string | null }

export async function getMe(): Promise<User> {
  return await request<User>('/v1/me')
}

export async function getCsrf(): Promise<{ token: string }> {
  return await request('/v1/auth/csrf')
}

export async function register(email: string, password: string): Promise<User> {
  return await request<User>('/v1/auth/register', { method: 'POST', json: { email, password } })
}

export async function login(email: string, password: string): Promise<{ user: User; token?: string }> {
  return await request('/v1/auth/login', { method: 'POST', json: { email, password } })
}

export async function logout(csrf: string): Promise<void> {
  await request<void>('/v1/auth/logout', { method: 'POST', headers: { 'X-CSRF-Token': csrf } })
}


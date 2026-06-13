// SECURITY NOTE: JWT access tokens are stored in localStorage for simplicity.
// This is vulnerable to XSS — any script running on the same origin can read the
// token and impersonate the user for up to 2 hours (the token TTL). Mitigations:
// - Backend validates JWT signature and checks a Redis blacklist on every request
// - Logout writes the token JTI to the blacklist
// - Security headers (CSP, X-Frame-Options) reduce XSS surface
// For production, consider migrating to httpOnly Secure SameSite cookies.
const TOKEN_KEY = 'task-platform:access-token'

export function getToken(): string | null {
  return window.localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  window.localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  window.localStorage.removeItem(TOKEN_KEY)
}

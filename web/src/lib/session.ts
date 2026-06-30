let cached: Promise<boolean> | undefined;

// isLoggedIn asks the server whether the (HttpOnly) session cookie is valid. The
// result is memoized so several islands on the same page share one request.
export function isLoggedIn(): Promise<boolean> {
  if (!cached) {
    cached = fetch('/api/auth/me', { headers: { Accept: 'application/json' } })
      .then((res) => res.ok)
      .catch(() => false);
  }
  return cached;
}

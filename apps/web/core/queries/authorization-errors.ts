export function isAuthorizationError(cause: unknown) {
  if (!cause || typeof cause !== "object") return false;
  const error = cause as { status?: number; response?: { status?: number } };
  return [401, 403, 404].includes(error.response?.status ?? error.status ?? 0);
}

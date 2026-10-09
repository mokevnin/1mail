export type ApiErrorLike = {
  detail?: string | null
  form?: string | null
  title?: string | null
  error?: string | null
  status?: number | null
}

function isApiErrorLike(error: unknown): error is ApiErrorLike {
  return typeof error === 'object' && error !== null
}

// A 403 problem+json body carries `status: 403` (RFC 7807); the backend answers
// it for role-gated actions a plain member is not allowed to perform.
export function isForbiddenError(error: unknown) {
  return isApiErrorLike(error) && error.status === 403
}

export function getApiErrorMessage(error: unknown, fallback: string) {
  if (!isApiErrorLike(error)) return fallback
  return error.detail ?? error.form ?? error.error ?? error.title ?? fallback
}

export type ApiErrorLike = {
  detail?: string | null
  form?: string | null
  title?: string | null
  error?: string | null
}

export function getApiErrorMessage(error: ApiErrorLike | null | undefined, fallback: string) {
  return error?.detail ?? error?.form ?? error?.error ?? error?.title ?? fallback
}

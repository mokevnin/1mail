import { useTranslation } from 'react-i18next'

import { getApiErrorMessage, isRateLimitedError } from '../utils/apiErrors.ts'

// The message to show for a failed API call: the localized "too many requests"
// text for a 429 (ADR 0018), the API detail otherwise, then the fallback.
export function useApiErrorMessage() {
  const { t } = useTranslation()
  return (error: unknown, fallback: string) =>
    isRateLimitedError(error)
      ? t(($) => $.notifications.rateLimited)
      : getApiErrorMessage(error, fallback)
}

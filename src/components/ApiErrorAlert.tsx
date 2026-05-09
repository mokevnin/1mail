import { Alert } from '@mantine/core'
import type { ReactNode } from 'react'
import { type ApiErrorLike, getApiErrorMessage } from '../utils/apiErrors.ts'

type ApiErrorAlertProps = {
  error: ApiErrorLike | null | undefined
  title: ReactNode
  fallback: string
}

export function ApiErrorAlert({ error, title, fallback }: ApiErrorAlertProps) {
  return (
    <Alert color="red" title={title}>
      {getApiErrorMessage(error, fallback)}
    </Alert>
  )
}

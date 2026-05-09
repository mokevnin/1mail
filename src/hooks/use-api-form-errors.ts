import type { UseMutationOptions } from '@tanstack/react-query'
import { useCallback } from 'react'

type ApiFormError = {
  errors?: Record<string, string[]>
}

type FormWithErrors = {
  setErrors: (errors: Record<string, string>) => void
}

function applyApiFormErrors(form: FormWithErrors, error: ApiFormError): boolean {
  if (!error.errors) {
    return false
  }

  const errors = Object.fromEntries(
    Object.entries(error.errors)
      .filter(([, messages]) => messages.length > 0)
      .map(([field, messages]) => [field, messages.join(', ')]),
  )

  if (Object.keys(errors).length === 0) {
    return false
  }

  form.setErrors(errors)
  return true
}

export function useApiFormErrors(form: FormWithErrors) {
  return useCallback(
    <TData, TError extends ApiFormError, TVariables, TOnMutateResult>(
      options: UseMutationOptions<TData, TError, TVariables, TOnMutateResult>,
    ): UseMutationOptions<TData, TError, TVariables, TOnMutateResult> => ({
      ...options,
      onError: (error, variables, onMutateResult, context) => {
        applyApiFormErrors(form, error)
        return options.onError?.(error, variables, onMutateResult, context)
      },
    }),
    [form],
  )
}

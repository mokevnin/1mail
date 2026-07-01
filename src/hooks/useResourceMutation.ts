import { notifications } from '@mantine/notifications'
import {
  type QueryKey,
  type UseMutationOptions,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { type ApiErrorLike, getApiErrorMessage } from '../utils/apiErrors.ts'

// Options layered on top of a generated mutation. The three things every CRUD
// mutation in this app repeats — invalidate the affected list query, toast on
// success, toast the API error on failure — collapse into declarative fields.
interface ResourceMutationOptions<TData, TError, TVars, TCtx> {
  // The generated react-query mutation, e.g. siteTemplatesCreateMutation().
  mutation: UseMutationOptions<TData, TError, TVars, TCtx>
  // Query keys to invalidate after a successful mutation (list queries, etc.).
  invalidate?: QueryKey[]
  // Toast message shown on success. Omit for no toast (the generic "Success"
  // title is supplied by the hook).
  successMessage?: string
  // Title used for the error toast. The message is the API error detail, or a
  // generic fallback when the API gives none — never a copy of the title.
  errorTitle: string
  // Extra success work — navigation, form reset — run after invalidation.
  onDone?: (data: TData, variables: TVars) => void | Promise<void>
}

// useResourceMutation wraps a generated mutation with the app's standard
// success/error UX so call sites stay declarative. It keeps react-query's typing
// by inferring TData/TError/TVars from the passed mutation options.
export function useResourceMutation<TData, TError, TVars, TCtx>({
  mutation,
  invalidate,
  successMessage,
  errorTitle,
  onDone,
}: ResourceMutationOptions<TData, TError, TVars, TCtx>) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  return useMutation({
    ...mutation,
    onSuccess: async (data, variables) => {
      await Promise.all(
        (invalidate ?? []).map((queryKey) => queryClient.invalidateQueries({ queryKey })),
      )
      if (successMessage) {
        notifications.show({
          color: 'teal',
          title: t(($) => $.notifications.successTitle),
          message: successMessage,
        })
      }
      await onDone?.(data, variables)
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: errorTitle,
        message: getApiErrorMessage(
          error as ApiErrorLike,
          t(($) => $.notifications.errorMessage),
        ),
      })
    },
  })
}

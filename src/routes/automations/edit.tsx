import { Loader, Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useEffectEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiErrorAlert } from '../../components/ApiErrorAlert.tsx'
import {
  siteAutomationsGetOptions,
  siteAutomationsGetQueryKey,
  siteAutomationsListQueryKey,
  siteAutomationsUpdateMutation,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import type { SiteAutomationResource } from '../../generated/site/types.gen.ts'
import { automationsEditRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import {
  AutomationForm,
  type AutomationFormValues,
  parseSteps,
  serializeSteps,
} from './AutomationForm.tsx'

export function AutomationEditPage() {
  const { t } = useTranslation()
  const { slug, automationId } = automationsEditRoute.useParams()
  const queryClient = useQueryClient()

  const form = useForm<AutomationFormValues>({
    initialValues: { name: '', triggerEvent: 'contact.created', steps: [] },
  })

  const getQuery = useQuery(
    siteAutomationsGetOptions({ path: { workspaceSlug: slug, id: automationId } }),
  )

  const applyData = useEffectEvent((data: SiteAutomationResource | undefined) => {
    if (!data) return
    form.setValues({
      name: data.name,
      triggerEvent: data.triggerEvent,
      steps: parseSteps(data.definition),
    })
  })

  useEffect(() => {
    applyData(getQuery.data)
  }, [getQuery.data])

  const updateMutation = useMutation({
    ...siteAutomationsUpdateMutation(),
    onSuccess: async (updated) => {
      await queryClient.invalidateQueries({
        queryKey: siteAutomationsListQueryKey({ path: { workspaceSlug: slug } }),
      })
      await queryClient.invalidateQueries({
        queryKey: siteAutomationsGetQueryKey({ path: { workspaceSlug: slug, id: updated.id } }),
      })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.automationUpdated),
      })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.alerts.automationSaveErrorTitle),
        message: getApiErrorMessage(
          error,
          t(($) => $.alerts.automationSaveErrorTitle),
        ),
      })
    },
  })

  if (getQuery.isLoading) return <Loader />

  if (getQuery.isError) {
    return (
      <ApiErrorAlert
        error={getQuery.error}
        title={t(($) => $.alerts.automationLoadErrorTitle)}
        fallback={t(($) => $.alerts.automationLoadErrorTitle)}
      />
    )
  }

  return (
    <Stack>
      <Title order={4}>{t(($) => $.automations.editTitle)}</Title>
      <AutomationForm
        form={form}
        isPending={updateMutation.isPending}
        onSubmit={(values) =>
          updateMutation.mutate({
            path: { workspaceSlug: slug, id: automationId },
            body: {
              name: values.name.trim(),
              triggerEvent: values.triggerEvent,
              definition: serializeSteps(values.steps),
            },
          })
        }
      />
    </Stack>
  )
}

import { Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteAutomationsCreateMutation,
  siteAutomationsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { automationsCreateRoute, automationsEditRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import { AutomationForm, type AutomationFormValues, serializeSteps } from './AutomationForm.tsx'

export function AutomationCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { slug } = automationsCreateRoute.useParams()

  const form = useForm<AutomationFormValues>({
    initialValues: { name: '', triggerEvent: 'contact.created', steps: [] },
  })

  const createMutation = useMutation({
    ...siteAutomationsCreateMutation(),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({
        queryKey: siteAutomationsListQueryKey({ path: { workspaceSlug: slug } }),
      })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.automationCreated),
      })
      await navigate({ to: automationsEditRoute.to, params: { slug, automationId: created.id } })
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

  return (
    <Stack>
      <Title order={4}>{t(($) => $.automations.createTitle)}</Title>
      <AutomationForm
        form={form}
        isPending={createMutation.isPending}
        onSubmit={(values) =>
          createMutation.mutate({
            path: { workspaceSlug: slug },
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

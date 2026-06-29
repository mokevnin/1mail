import { Button, Group, Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteAutomationsCreateMutation,
  siteAutomationsListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { automationsCreateRoute, automationsEditRoute, automationsRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import {
  AutomationDetailsFields,
  type AutomationDetailsValues,
} from './AutomationDetailsFields.tsx'

// Create captures only name + trigger, then redirects into the visual builder
// (edit) where the flow itself is designed — so the builder always has an id.
export function AutomationCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { slug } = automationsCreateRoute.useParams()

  const form = useForm<AutomationDetailsValues>({
    initialValues: { name: '', triggerEvent: 'contact.created' },
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
      <form
        onSubmit={form.onSubmit((values) =>
          createMutation.mutate({
            path: { workspaceSlug: slug },
            body: { name: values.name.trim(), triggerEvent: values.triggerEvent },
          }),
        )}
      >
        <Stack>
          <AutomationDetailsFields form={form} />
          <Group justify="flex-end">
            <Button
              variant="default"
              type="button"
              onClick={() => navigate({ to: automationsRoute.to, params: { slug } })}
            >
              {t(($) => $.actions.cancel)}
            </Button>
            <Button type="submit" loading={createMutation.isPending}>
              {t(($) => $.actions.continue)}
            </Button>
          </Group>
        </Stack>
      </form>
    </Stack>
  )
}

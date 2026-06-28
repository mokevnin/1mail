import { Stack, Title } from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import {
  siteTemplatesCreateMutation,
  siteTemplatesListQueryKey,
} from '../../generated/site/@tanstack/react-query.gen.ts'
import { templatesCreateRoute, templatesEditRoute } from '../../router.tsx'
import { getApiErrorMessage } from '../../utils/apiErrors.ts'
import { TemplateForm, type TemplateFormValues } from './TemplateForm.tsx'

export function TemplateCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { slug } = templatesCreateRoute.useParams()

  const form = useForm<TemplateFormValues>({
    initialValues: { name: '', subject: '', body: '' },
  })

  const createMutation = useMutation({
    ...siteTemplatesCreateMutation(),
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({
        queryKey: siteTemplatesListQueryKey({ path: { workspaceSlug: slug } }),
      })
      notifications.show({
        color: 'teal',
        title: t(($) => $.notifications.successTitle),
        message: t(($) => $.notifications.templateCreated),
      })
      await navigate({ to: templatesEditRoute.to, params: { slug, templateId: created.id } })
    },
    onError: (error) => {
      notifications.show({
        color: 'red',
        title: t(($) => $.alerts.templateSaveErrorTitle),
        message: getApiErrorMessage(
          error,
          t(($) => $.alerts.templateSaveErrorTitle),
        ),
      })
    },
  })

  return (
    <Stack>
      <Title order={4}>{t(($) => $.templates.createTitle)}</Title>
      <TemplateForm
        form={form}
        isPending={createMutation.isPending}
        onSubmit={(values) =>
          createMutation.mutate({
            path: { workspaceSlug: slug },
            body: {
              name: values.name.trim(),
              subject: values.subject.trim(),
              body: values.body,
            },
          })
        }
      />
    </Stack>
  )
}

import { Button, Group, Stack, Textarea, TextInput } from '@mantine/core'
import type { useForm } from '@mantine/form'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { templatesRoute } from '../../router.tsx'

export interface TemplateFormValues {
  name: string
  subject: string
  body: string
}

type TemplateFormInstance = ReturnType<typeof useForm<TemplateFormValues>>

interface TemplateFormProps {
  form: TemplateFormInstance
  isPending: boolean
  onSubmit: (values: TemplateFormValues) => void
}

export function TemplateForm({ form, isPending, onSubmit }: TemplateFormProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = useParams({ strict: false })

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput
          label={t(($) => $.templates.nameLabel)}
          required
          {...form.getInputProps('name')}
        />
        <TextInput label={t(($) => $.templates.subjectLabel)} {...form.getInputProps('subject')} />
        <Textarea
          label={t(($) => $.templates.bodyLabel)}
          description={t(($) => $.templates.bodyHint)}
          autosize
          minRows={12}
          {...form.getInputProps('body')}
        />

        <Group justify="flex-end">
          <Button
            variant="default"
            type="button"
            onClick={() => slug && navigate({ to: templatesRoute.to, params: { slug } })}
          >
            {t(($) => $.actions.cancel)}
          </Button>
          <Button type="submit" loading={isPending}>
            {t(($) => $.actions.save)}
          </Button>
        </Group>
      </Stack>
    </form>
  )
}

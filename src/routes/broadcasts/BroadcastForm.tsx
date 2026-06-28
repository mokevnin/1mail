import { Alert, Button, Group, Input, Select, Stack, TextInput } from '@mantine/core'
import type { useForm } from '@mantine/form'
import { useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { RichTextField } from '../../components/RichTextField.tsx'
import { siteSegmentsListOptions } from '../../generated/site/@tanstack/react-query.gen.ts'
import { broadcastsRoute } from '../../router.tsx'

export interface BroadcastFormValues {
  name: string
  subject: string
  fromName: string
  fromEmail: string
  bodyHtml: string
  segmentId: string
}

type BroadcastFormInstance = ReturnType<typeof useForm<BroadcastFormValues>>

interface BroadcastFormProps {
  form: BroadcastFormInstance
  isPending: boolean
  onSubmit: (values: BroadcastFormValues) => void
}

export function BroadcastForm({ form, isPending, onSubmit }: BroadcastFormProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = useParams({ strict: false })

  const segmentsQuery = useQuery({
    ...siteSegmentsListOptions({ path: { workspaceSlug: slug ?? '' }, query: { pageSize: 100 } }),
    enabled: Boolean(slug),
  })
  const segmentOptions = [
    { value: '', label: t(($) => $.broadcasts.audienceAll) },
    ...(segmentsQuery.data?.items ?? [])
      .filter((s) => s.type === 'rule')
      .map((s) => ({ value: s.id, label: s.name })),
  ]

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput
          label={t(($) => $.broadcasts.nameLabel)}
          required
          {...form.getInputProps('name')}
        />
        <TextInput label={t(($) => $.broadcasts.subjectLabel)} {...form.getInputProps('subject')} />
        <Group grow align="flex-start">
          <TextInput
            label={t(($) => $.broadcasts.fromNameLabel)}
            {...form.getInputProps('fromName')}
          />
          <TextInput
            label={t(($) => $.broadcasts.fromEmailLabel)}
            type="email"
            {...form.getInputProps('fromEmail')}
          />
        </Group>

        <Select
          label={t(($) => $.broadcasts.audienceLabel)}
          data={segmentOptions}
          allowDeselect={false}
          {...form.getInputProps('segmentId')}
        />
        <Alert color="blue" variant="light">
          {t(($) => $.broadcasts.audienceNote)}
        </Alert>

        <Input.Wrapper label={t(($) => $.broadcasts.bodyLabel)}>
          <RichTextField
            value={form.values.bodyHtml}
            onChange={(html) => form.setFieldValue('bodyHtml', html)}
          />
        </Input.Wrapper>

        <Group justify="flex-end">
          <Button
            variant="default"
            type="button"
            onClick={() => slug && navigate({ to: broadcastsRoute.to, params: { slug } })}
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

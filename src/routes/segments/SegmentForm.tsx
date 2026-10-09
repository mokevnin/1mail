import { Button, Group, Input, Stack, TextInput } from '@mantine/core'
import type { useForm } from '@mantine/form'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { segmentsRoute } from '../../router.tsx'
import { SegmentRuleBuilder } from './SegmentRuleBuilder.tsx'

export interface SegmentFormValues {
  name: string
  definition: string
}

type SegmentFormInstance = ReturnType<typeof useForm<SegmentFormValues>>

interface SegmentFormProps {
  form: SegmentFormInstance
  isPending: boolean
  onSubmit: (values: SegmentFormValues) => void
}

export function SegmentForm({ form, isPending, onSubmit }: SegmentFormProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = useParams({ strict: false })

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput
          label={t(($) => $.segments.nameLabel)}
          required
          {...form.getInputProps('name')}
        />
        {slug ? (
          <Input.Wrapper
            label={t(($) => $.segments.rulesLabel)}
            description={t(($) => $.segments.definitionHint)}
          >
            <SegmentRuleBuilder
              slug={slug}
              value={form.values.definition}
              onChange={(json) => form.setFieldValue('definition', json)}
            />
          </Input.Wrapper>
        ) : null}

        <Group justify="flex-end">
          <Button
            variant="default"
            type="button"
            onClick={() => slug && navigate({ to: segmentsRoute.to, params: { slug } })}
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

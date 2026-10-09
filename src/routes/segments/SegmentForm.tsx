import { Button, Group, Input, Stack, TextInput } from '@mantine/core'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import type { ResourceFormProps } from '../../resources/defineResource.tsx'
import { segmentsRoute } from '../../router.tsx'
import { SegmentRuleBuilder } from './SegmentRuleBuilder.tsx'

// A Segment is always a rule, so the payload carries a definition string.
export type SegmentPayload = { name: string; definition: string }

export function SegmentForm({ form, isPending, onSubmit }: ResourceFormProps<SegmentPayload>) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = useParams({ strict: false })

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput
          label={t(($) => $.segments.nameLabel)}
          withAsterisk
          {...form.getInputProps('name')}
        />
        {slug ? (
          <Input.Wrapper
            label={t(($) => $.segments.rulesLabel)}
            description={t(($) => $.segments.definitionHint)}
          >
            <SegmentRuleBuilder
              slug={slug}
              value={form.values.definition ?? ''}
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

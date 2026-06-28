import { Button, Group, Stack, Text } from '@mantine/core'
import { QueryBuilderMantine } from '@react-querybuilder/mantine'
import { useMutation } from '@tanstack/react-query'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { type Field, type Operator, QueryBuilder, type RuleGroupType } from 'react-querybuilder'
import { siteSegmentsPreviewMutation } from '../../generated/site/@tanstack/react-query.gen.ts'

// Operators limited to what the backend segment engine compiles.
const operators: Operator[] = [
  { name: '=', label: 'is' },
  { name: '!=', label: 'is not' },
  { name: 'contains', label: 'contains' },
  { name: 'beginsWith', label: 'begins with' },
  { name: 'null', label: 'is empty' },
  { name: 'notNull', label: 'is set' },
]

const fields: Field[] = [
  { name: 'email', label: 'Email' },
  { name: 'first_name', label: 'First name' },
  { name: 'last_name', label: 'Last name' },
  { name: 'time_zone', label: 'Time zone' },
  {
    name: 'status',
    label: 'Status',
    valueEditorType: 'select',
    values: [
      { name: 'active', label: 'Active' },
      { name: 'unsubscribed', label: 'Unsubscribed' },
    ],
  },
]

const emptyQuery: RuleGroupType = { combinator: 'and', rules: [] }

function parseQuery(def: string): RuleGroupType {
  if (!def.trim()) return emptyQuery
  try {
    return JSON.parse(def) as RuleGroupType
  } catch {
    return emptyQuery
  }
}

interface SegmentRuleBuilderProps {
  slug: string
  value: string
  onChange: (json: string) => void
}

// SegmentRuleBuilder is the visual rule editor (react-querybuilder + the official
// Mantine compat package — no hand-rolled condition rows, no custom CSS). It emits
// the react-querybuilder RuleGroupType JSON the backend engine consumes directly,
// and offers a live preview of how many active contacts match.
export function SegmentRuleBuilder({ slug, value, onChange }: SegmentRuleBuilderProps) {
  const { t } = useTranslation()
  const query = useMemo(() => parseQuery(value), [value])
  const preview = useMutation(siteSegmentsPreviewMutation())

  return (
    <Stack gap="xs">
      <QueryBuilderMantine>
        <QueryBuilder
          fields={fields}
          operators={operators}
          query={query}
          onQueryChange={(q) => onChange(JSON.stringify(q))}
        />
      </QueryBuilderMantine>
      <Group>
        <Button
          variant="light"
          size="compact-sm"
          loading={preview.isPending}
          onClick={() =>
            preview.mutate({ path: { workspaceSlug: slug }, body: { definition: value } })
          }
        >
          {t(($) => $.segments.previewButton)}
        </Button>
        {preview.data ? (
          <Text size="sm">{t(($) => $.segments.previewCount, { count: preview.data.count })}</Text>
        ) : null}
      </Group>
    </Stack>
  )
}

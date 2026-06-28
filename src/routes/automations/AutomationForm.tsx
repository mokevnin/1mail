import {
  ActionIcon,
  Button,
  Card,
  Group,
  NumberInput,
  Select,
  Stack,
  Text,
  Textarea,
  TextInput,
  Title,
} from '@mantine/core'
import type { useForm } from '@mantine/form'
import { IconArrowDown, IconArrowUp, IconTrash } from '@tabler/icons-react'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { automationsRoute } from '../../router.tsx'

// Step mirrors the engine's step JSON ({type, subject, body, seconds}) so the
// form value serializes straight into the automation definition.
export interface AutomationStep {
  type: 'email' | 'wait'
  subject: string
  body: string
  seconds: number
}

export interface AutomationFormValues {
  name: string
  triggerEvent: string
  steps: AutomationStep[]
}

type AutomationFormInstance = ReturnType<typeof useForm<AutomationFormValues>>

interface AutomationFormProps {
  form: AutomationFormInstance
  isPending: boolean
  onSubmit: (values: AutomationFormValues) => void
}

export function emptyEmailStep(): AutomationStep {
  return { type: 'email', subject: '', body: '', seconds: 0 }
}

export function emptyWaitStep(): AutomationStep {
  return { type: 'wait', subject: '', body: '', seconds: 3600 }
}

export function AutomationForm({ form, isPending, onSubmit }: AutomationFormProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = useParams({ strict: false })

  const triggerOptions = [
    { value: 'contact.created', label: t(($) => $.automations.triggerContactCreated) },
    { value: 'email.opened', label: t(($) => $.automations.triggerEmailOpened) },
    { value: 'email.clicked', label: t(($) => $.automations.triggerEmailClicked) },
    { value: 'email.unsubscribed', label: t(($) => $.automations.triggerEmailUnsubscribed) },
  ]

  const steps = form.getValues().steps

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput
          label={t(($) => $.automations.nameLabel)}
          required
          {...form.getInputProps('name')}
        />
        <Select
          label={t(($) => $.automations.triggerLabel)}
          description={t(($) => $.automations.triggerHint)}
          data={triggerOptions}
          required
          allowDeselect={false}
          {...form.getInputProps('triggerEvent')}
        />

        <Title order={5}>{t(($) => $.automations.stepsLabel)}</Title>

        {steps.length === 0 ? <Text c="dimmed">{t(($) => $.automations.noSteps)}</Text> : null}

        <Stack gap="sm">
          {steps.map((step, index) => (
            <Card key={index} withBorder padding="sm">
              <Stack gap="xs">
                <Group justify="space-between">
                  <Text fw={600}>
                    {index + 1}.{' '}
                    {step.type === 'email'
                      ? t(($) => $.automations.stepEmail)
                      : t(($) => $.automations.stepWait)}
                  </Text>
                  <Group gap={4}>
                    <ActionIcon
                      variant="subtle"
                      aria-label={t(($) => $.automations.moveUp)}
                      disabled={index === 0}
                      onClick={() => form.reorderListItem('steps', { from: index, to: index - 1 })}
                    >
                      <IconArrowUp size={16} />
                    </ActionIcon>
                    <ActionIcon
                      variant="subtle"
                      aria-label={t(($) => $.automations.moveDown)}
                      disabled={index === steps.length - 1}
                      onClick={() => form.reorderListItem('steps', { from: index, to: index + 1 })}
                    >
                      <IconArrowDown size={16} />
                    </ActionIcon>
                    <ActionIcon
                      variant="subtle"
                      color="red"
                      aria-label={t(($) => $.automations.removeStep)}
                      onClick={() => form.removeListItem('steps', index)}
                    >
                      <IconTrash size={16} />
                    </ActionIcon>
                  </Group>
                </Group>

                {step.type === 'email' ? (
                  <>
                    <TextInput
                      label={t(($) => $.automations.emailSubjectLabel)}
                      required
                      {...form.getInputProps(`steps.${index}.subject`)}
                    />
                    <Textarea
                      label={t(($) => $.automations.emailBodyLabel)}
                      description={t(($) => $.automations.emailBodyHint)}
                      autosize
                      minRows={6}
                      {...form.getInputProps(`steps.${index}.body`)}
                    />
                  </>
                ) : (
                  <NumberInput
                    label={t(($) => $.automations.waitSecondsLabel)}
                    min={1}
                    {...form.getInputProps(`steps.${index}.seconds`)}
                  />
                )}
              </Stack>
            </Card>
          ))}
        </Stack>

        <Group>
          <Button
            variant="light"
            type="button"
            onClick={() => form.insertListItem('steps', emptyEmailStep())}
          >
            {t(($) => $.automations.addEmailStep)}
          </Button>
          <Button
            variant="light"
            type="button"
            onClick={() => form.insertListItem('steps', emptyWaitStep())}
          >
            {t(($) => $.automations.addWaitStep)}
          </Button>
        </Group>

        <Group justify="flex-end">
          <Button
            variant="default"
            type="button"
            onClick={() => slug && navigate({ to: automationsRoute.to, params: { slug } })}
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

// serializeSteps turns the form steps into the JSON definition the engine reads,
// dropping fields irrelevant to each step type.
export function serializeSteps(steps: AutomationStep[]): string {
  return JSON.stringify(
    steps.map((s) =>
      s.type === 'email'
        ? { type: 'email', subject: s.subject, body: s.body }
        : // NumberInput yields '' when cleared; coerce so the JSON carries an int.
          { type: 'wait', seconds: Math.trunc(Number(s.seconds)) || 0 },
    ),
  )
}

// parseSteps reads a stored definition back into editable form steps, tolerating
// an empty or malformed value (new automations default to "[]").
export function parseSteps(definition: string): AutomationStep[] {
  try {
    const raw = JSON.parse(definition || '[]')
    if (!Array.isArray(raw)) return []
    return raw.map((s: Partial<AutomationStep>) => ({
      type: s.type === 'wait' ? 'wait' : 'email',
      subject: s.subject ?? '',
      body: s.body ?? '',
      seconds: s.seconds ?? 3600,
    }))
  } catch {
    return []
  }
}

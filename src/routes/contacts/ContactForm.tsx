import { Button, Group, Stack, TextInput } from '@mantine/core'
import type { useForm } from '@mantine/form'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import type { SiteCreateContactInput } from '../../generated/site/types.gen.ts'
import { contactsRoute } from '../../router.tsx'

// Identity is multi-key: subject_id / email / phone are alias keys, any of which may
// be absent (ADR 0002). The form keeps each as a string and omits the empty ones
// from the payload so a blank email is not sent (and fails email-format validation).
export type ContactFormValues = Record<
  keyof Pick<
    SiteCreateContactInput,
    'subjectId' | 'email' | 'phone' | 'firstName' | 'lastName' | 'timeZone'
  >,
  string
>

// toContactPayload trims values and omits the empty ones, so optional alias keys are
// sent only when set (respecting exactOptionalPropertyTypes and email validation).
export function toContactPayload(v: ContactFormValues): SiteCreateContactInput {
  const subjectId = v.subjectId.trim()
  const email = v.email.trim()
  const phone = v.phone.trim()
  const firstName = v.firstName.trim()
  const lastName = v.lastName.trim()
  const timeZone = v.timeZone.trim()
  return {
    ...(subjectId && { subjectId }),
    ...(email && { email }),
    ...(phone && { phone }),
    ...(firstName && { firstName }),
    ...(lastName && { lastName }),
    ...(timeZone && { timeZone }),
  }
}

type ContactFormInstance = ReturnType<typeof useForm<ContactFormValues>>

interface ContactFormProps {
  form: ContactFormInstance
  isPending: boolean
  onSubmit: (values: ContactFormValues) => void
}

export function ContactForm({ form, isPending, onSubmit }: ContactFormProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = useParams({ strict: false })

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput label={t(($) => $.table.email)} {...form.getInputProps('email')} />
        <TextInput label={t(($) => $.table.subjectId)} {...form.getInputProps('subjectId')} />
        <TextInput label={t(($) => $.table.phone)} {...form.getInputProps('phone')} />
        <TextInput label={t(($) => $.table.firstName)} {...form.getInputProps('firstName')} />
        <TextInput label={t(($) => $.table.lastName)} {...form.getInputProps('lastName')} />
        <TextInput label={t(($) => $.table.timeZone)} {...form.getInputProps('timeZone')} />

        <Group justify="flex-end">
          <Button
            variant="default"
            type="button"
            onClick={() => slug && navigate({ to: contactsRoute.to, params: { slug } })}
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

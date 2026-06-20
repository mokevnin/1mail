import { Button, Group, Stack, TextInput } from '@mantine/core'
import type { useForm } from '@mantine/form'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import type { SiteCreateContactInput } from '../../generated/site/types.gen.ts'
import { contactsRoute } from '../../router.tsx'

export type ContactFormValues = Record<
  keyof Pick<SiteCreateContactInput, 'email' | 'firstName' | 'lastName' | 'timeZone'>,
  string
>

type ContactFormInstance = ReturnType<typeof useForm<ContactFormValues>>

interface ContactFormProps {
  form: ContactFormInstance
  emailEditable?: boolean
  isPending: boolean
  onSubmit: (values: ContactFormValues) => void
}

export function ContactForm({ form, emailEditable = true, isPending, onSubmit }: ContactFormProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { slug } = useParams({ strict: false })

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput
          label="Email"
          required={emailEditable}
          disabled={!emailEditable}
          {...form.getInputProps('email')}
        />
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

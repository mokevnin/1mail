import { Button, Group, Stack, TextInput } from '@mantine/core'
import { useForm } from '@mantine/form'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { contactsRoute } from '../../router.tsx'
import type { ContactFormValues } from './form.ts'

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
          <Button variant="default" type="button" onClick={() => navigate({ to: contactsRoute.to })}>
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

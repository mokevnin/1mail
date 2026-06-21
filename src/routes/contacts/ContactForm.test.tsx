import { useForm } from '@mantine/form'
import { expect, test, vi } from 'vitest'
import { renderWithRouter } from '../../test/renderWithRouter.tsx'
import { ContactForm, type ContactFormValues } from './ContactForm.tsx'

// ContactForm receives a Mantine form as a prop, so the harness builds a real
// one and renders the form under it — same shape as create.tsx / edit.tsx.
function Harness({
  emailEditable = true,
  onSubmit,
}: {
  emailEditable?: boolean
  onSubmit: (values: ContactFormValues) => void
}) {
  const form = useForm<ContactFormValues>({
    initialValues: { email: '', firstName: '', lastName: '', timeZone: '' },
  })
  return (
    <ContactForm form={form} emailEditable={emailEditable} isPending={false} onSubmit={onSubmit} />
  )
}

test('submits the entered values', async () => {
  const onSubmit = vi.fn()
  const { screen } = await renderWithRouter(<Harness onSubmit={onSubmit} />)

  await screen.getByLabelText('Email').fill('ada@example.com')
  await screen.getByLabelText('First name').fill('Ada')
  await screen.getByLabelText('Last name').fill('Lovelace')
  await screen.getByRole('button', { name: 'Save' }).click()

  // Mantine's form.onSubmit invokes the handler with (values, event).
  await vi.waitFor(() =>
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ email: 'ada@example.com', firstName: 'Ada', lastName: 'Lovelace' }),
      expect.anything(),
    ),
  )
})

test('disables the email field when not editable', async () => {
  const { screen } = await renderWithRouter(<Harness emailEditable={false} onSubmit={vi.fn()} />)

  await expect.element(screen.getByLabelText('Email')).toBeDisabled()
})

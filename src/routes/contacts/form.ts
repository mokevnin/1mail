export type ContactFormValues = {
  email: string
  firstName: string
  lastName: string
  timeZone: string
}

export const EMPTY_CONTACT_FORM: ContactFormValues = {
  email: '',
  firstName: '',
  lastName: '',
  timeZone: '',
}

export function toCreateNullableField(value: string): string | null | undefined {
  const trimmed = value.trim()
  return trimmed === '' ? undefined : trimmed
}

export function toUpdateNullableField(value: string): string | null {
  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

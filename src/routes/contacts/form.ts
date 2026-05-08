import type { SiteCreateContactInput } from '../../generated/site/types.gen.ts'

export type ContactFormValues = Record<
  keyof Pick<SiteCreateContactInput, 'email' | 'firstName' | 'lastName' | 'timeZone'>,
  string
>

export const EMPTY_CONTACT_FORM: ContactFormValues = {
  email: '',
  firstName: '',
  lastName: '',
  timeZone: '',
}

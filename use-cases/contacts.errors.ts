import { TaggedError } from 'better-result'

export class ContactAlreadyExistsError extends TaggedError('ContactAlreadyExistsError')<{
  message: string
  email: string
}>() {
  constructor(args: { email: string }) {
    super({
      email: args.email,
      message: 'Contact with this email already exists',
    })
  }
}

export class ContactNotFoundError extends TaggedError('ContactNotFoundError')<{
  message: string
  id: number
}>() {
  constructor(args: { id: number }) {
    super({
      id: args.id,
      message: `Contact with id=${args.id} was not found`,
    })
  }
}

export class ContactCreateFailedError extends TaggedError('ContactCreateFailedError')<{
  message: string
}>() {
  constructor() {
    super({ message: 'Failed to create contact' })
  }
}

export type ContactCreateError = ContactAlreadyExistsError | ContactCreateFailedError
export type ContactUpdateError = ContactNotFoundError

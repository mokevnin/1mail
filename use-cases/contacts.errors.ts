import { TaggedError } from 'better-result'

export class ContactAlreadyExistsError extends TaggedError('ContactAlreadyExistsError')<{
  code: 'conflict'
  message: string
  email: string
}>() {
  constructor(args: { email: string }) {
    super({
      code: 'conflict',
      email: args.email,
      message: 'Contact with this email already exists',
    })
  }
}

export class ContactNotFoundError extends TaggedError('ContactNotFoundError')<{
  code: 'not_found'
  message: string
  id: bigint
}>() {
  constructor(args: { id: bigint }) {
    super({
      code: 'not_found',
      id: args.id,
      message: `Contact with id=${args.id} was not found`,
    })
  }
}

export class ContactCreateFailedError extends TaggedError('ContactCreateFailedError')<{
  code: 'internal'
  message: string
}>() {
  constructor() {
    super({
      code: 'internal',
      message: 'Failed to create contact',
    })
  }
}

export type ContactCreateError = ContactAlreadyExistsError | ContactCreateFailedError
export type ContactUpdateError = ContactNotFoundError

import { Result } from 'better-result'
import { eq } from 'drizzle-orm'

import { type AppDatabase, isUniqueViolation } from '../db/runtime.ts'
import { type ContactRecord, contacts } from '../db/schema.ts'
import type { CreateContactInput, UpdateContactInput } from '../types/contacts.ts'
import {
  ContactAlreadyExistsError,
  type ContactCreateError,
  ContactCreateFailedError,
  ContactNotFoundError,
  type ContactUpdateError,
} from './contacts.errors.ts'

export async function createContact(
  db: AppDatabase,
  input: CreateContactInput,
): Promise<Result<ContactRecord, ContactCreateError>> {
  try {
    const [created] = await db.insert(contacts).values(input).returning()

    if (!created) {
      return Result.err(new ContactCreateFailedError())
    }

    return Result.ok(created)
  } catch (error) {
    if (isUniqueViolation(error)) {
      return Result.err(new ContactAlreadyExistsError({ email: input.email }))
    }

    throw error
  }
}

export async function updateContact(
  db: AppDatabase,
  id: bigint,
  input: UpdateContactInput,
): Promise<Result<ContactRecord, ContactUpdateError>> {
  const [updated] = await db.update(contacts).set(input).where(eq(contacts.id, id)).returning()

  if (!updated) {
    return Result.err(new ContactNotFoundError({ id }))
  }

  return Result.ok(updated)
}

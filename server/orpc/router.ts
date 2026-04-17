import { implement, ORPCError } from '@orpc/server'
import { asc, eq } from 'drizzle-orm'

import type { AppDatabase } from '#/db/runtime.ts'
import { contacts } from '#/db/schema.ts'
import { contract } from '#/generated/site/orpc.gen.ts'
import { zSiteContactsCreateBodyInput, zSiteContactsUpdateBodyInput } from '#/lib/http-zod.ts'
import { paginateQuery } from '#/lib/pagination.ts'
import { toContactResource } from '#/resources/contacts.ts'
import { createContact, updateContact } from '#/use-cases/contacts.ts'

export type OrpcContext = {
  db: AppDatabase
}

function throwNotFound(): never {
  throw new ORPCError('NOT_FOUND')
}

function throwUseCaseError(error: { code: string; message: string }): never {
  if (error.code === 'conflict') {
    throw new ORPCError('CONFLICT', {
      message: error.message,
    })
  }

  if (error.code === 'not_found') {
    throw new ORPCError('NOT_FOUND', {
      message: error.message,
    })
  }

  throw new ORPCError('INTERNAL_SERVER_ERROR', {
    message: error.message,
  })
}

const os = implement(contract).$context<OrpcContext>()

export const router = os.router({
  siteContactsList: os.siteContactsList.handler(async ({ context, input }) => {
    const whereClause = input.query?.status ? eq(contacts.status, input.query.status) : undefined

    return paginateQuery({
      db: context.db,
      pagination: input.query,
      table: contacts,
      query: (table) => context.db.select().from(table).where(whereClause).orderBy(asc(table.id)),
      map: toContactResource,
    })
  }),

  siteContactsCreate: os.siteContactsCreate.handler(async ({ context, input }) => {
    const body = zSiteContactsCreateBodyInput.parse(input.body)
    const createResult = await createContact(context.db, body)
    const result = createResult.map(toContactResource)

    if (result.isErr()) {
      return throwUseCaseError(result.error)
    }

    return result.value
  }),

  siteContactsDelete: os.siteContactsDelete.handler(async ({ context, input }) => {
    const [deleted] = await context.db
      .delete(contacts)
      .where(eq(contacts.id, BigInt(input.params.id)))
      .returning()

    if (!deleted) {
      return throwNotFound()
    }
  }),

  siteContactsGet: os.siteContactsGet.handler(async ({ context, input }) => {
    const [contact] = await context.db
      .select()
      .from(contacts)
      .where(eq(contacts.id, BigInt(input.params.id)))
      .limit(1)

    if (!contact) {
      return throwNotFound()
    }

    return toContactResource(contact)
  }),

  siteContactsUpdate: os.siteContactsUpdate.handler(async ({ context, input }) => {
    const body = zSiteContactsUpdateBodyInput.parse(input.body)
    const updateResult = await updateContact(context.db, BigInt(input.params.id), body)
    const result = updateResult.map(toContactResource)

    if (result.isErr()) {
      return throwUseCaseError(result.error)
    }

    return result.value
  }),
})

export type OrpcRouter = typeof router

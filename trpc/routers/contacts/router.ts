import { asc, eq } from 'drizzle-orm'
import { z } from 'zod'

import { contacts } from '../../../db/schema.ts'
import {
  zContactsCreateBody,
  zContactsDeletePath,
  zContactsGetPath,
  zContactsListQuery,
  zContactsUpdateBody,
  zContactsUpdatePath,
} from '../../../generated/handlers/zod.gen.ts'
import { paginateQuery } from '../../../lib/pagination.ts'
import { toContactResource } from '../../../resources/contacts.ts'
import { createContact, updateContact } from '../../../use-cases/contacts.ts'
import { throwTrpcError } from '../../errors.ts'
import { t } from '../../init.ts'
import { trpcGuard } from '../../utils.ts'

const zContactsUpdateInput = z.object({
  id: zContactsUpdatePath.shape.id,
  data: zContactsUpdateBody,
})

export const contactsRouter = t.router({
  list: t.procedure.input(zContactsListQuery.partial().optional()).query(async ({ ctx, input }) => {
    const whereClause = input?.status ? eq(contacts.status, input.status) : undefined

    return paginateQuery({
      db: ctx.db,
      pagination: input,
      table: contacts,
      query: (table) => ctx.db.select().from(table).where(whereClause).orderBy(asc(table.id)),
      map: toContactResource,
    })
  }),

  get: t.procedure.input(zContactsGetPath).query(async ({ ctx, input }) => {
    const [contact] = await ctx.db
      .select()
      .from(contacts)
      .where(eq(contacts.id, BigInt(input.id)))
      .limit(1)

    trpcGuard.found(contact)

    return toContactResource(contact)
  }),

  create: t.procedure.input(zContactsCreateBody).mutation(async ({ ctx, input }) => {
    const respond = (await createContact(ctx.db, input)).map(toContactResource).match({
      ok: (contact) => () => contact,
      err: (error) => () => throwTrpcError(error),
    })

    return respond()
  }),

  update: t.procedure.input(zContactsUpdateInput).mutation(async ({ ctx, input }) => {
    const respond = (await updateContact(ctx.db, BigInt(input.id), input.data))
      .map(toContactResource)
      .match({
        ok: (contact) => () => contact,
        err: (error) => () => throwTrpcError(error),
      })

    return respond()
  }),

  delete: t.procedure.input(zContactsDeletePath).mutation(async ({ ctx, input }) => {
    const [deleted] = await ctx.db
      .delete(contacts)
      .where(eq(contacts.id, BigInt(input.id)))
      .returning()

    trpcGuard.found(deleted)

    return { success: true }
  }),
})

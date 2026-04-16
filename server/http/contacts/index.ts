import { asc, eq } from 'drizzle-orm'
import type { FastifyInstance, FastifyPluginAsync } from 'fastify'

import { contacts } from '../../../db/schema.ts'
import type { RouteHandlers } from '../../../generated/handlers/fastify.gen.ts'
import { toFastifySchema } from '../../../lib/openapi.ts'
import { toContactResource } from '../../../resources/contacts.ts'
import { createContact, updateContact } from '../../../use-cases/contacts.ts'

const contactsPlugin: FastifyPluginAsync = async (
  fastify: FastifyInstance,
  _opts,
): Promise<void> => {
  const handlers = {
    contactsList: async (request, reply) => {
      const whereClause = request.query?.status
        ? eq(contacts.status, request.query.status)
        : undefined

      const response = await request.paginate({
        table: contacts,
        query: (table) => fastify.db.select().from(table).where(whereClause).orderBy(asc(table.id)),
        map: toContactResource,
      })

      return reply.code(200).send(response)
    },

    contactsCreate: async (request, reply) => {
      const createResult = await createContact(fastify.db, request.body)
      const result = createResult.map(toContactResource)

      if (result.isErr()) {
        return fastify.app.throwHttpError(result.error)
      }

      return reply.code(201).send(result.value)
    },

    contactsGet: async (request, reply) => {
      const id = BigInt(request.params.id)
      const [contact] = await fastify.db.select().from(contacts).where(eq(contacts.id, id)).limit(1)

      fastify.app.guard.found(contact)

      return reply.code(200).send(toContactResource(contact))
    },

    contactsUpdate: async (request, reply) => {
      const id = BigInt(request.params.id)
      const updateResult = await updateContact(fastify.db, id, request.body)
      const result = updateResult.map(toContactResource)

      if (result.isErr()) {
        return fastify.app.throwHttpError(result.error)
      }

      return reply.code(200).send(result.value)
    },

    contactsDelete: async (request, reply) => {
      const id = BigInt(request.params.id)
      const [deleted] = await fastify.db.delete(contacts).where(eq(contacts.id, id)).returning()

      fastify.app.guard.found(deleted)

      return reply.code(204).send()
    },
  } satisfies Pick<
    RouteHandlers,
    'contactsList' | 'contactsCreate' | 'contactsGet' | 'contactsUpdate' | 'contactsDelete'
  >

  fastify.get(
    '/',
    {
      schema: toFastifySchema('/contacts', 'GET'),
    },
    handlers.contactsList,
  )
  fastify.post(
    '/',
    {
      schema: toFastifySchema('/contacts', 'POST'),
    },
    handlers.contactsCreate,
  )
  fastify.get(
    '/:id',
    {
      schema: toFastifySchema('/contacts/{id}', 'GET'),
    },
    handlers.contactsGet,
  )
  fastify.put(
    '/:id',
    {
      schema: toFastifySchema('/contacts/{id}', 'PUT'),
    },
    handlers.contactsUpdate,
  )
  fastify.delete(
    '/:id',
    {
      schema: toFastifySchema('/contacts/{id}', 'DELETE'),
    },
    handlers.contactsDelete,
  )
}

export default contactsPlugin

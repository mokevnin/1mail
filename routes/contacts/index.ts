import type { FastifyPluginAsyncTypebox } from '@fastify/type-provider-typebox'
import { asc, eq } from 'drizzle-orm'
import type { FastifyInstance } from 'fastify'

import { contacts } from '../../db/schema.ts'
import type { RouteHandlers } from '../../generated/handlers/fastify.gen.ts'
import type { ContactPage } from '../../generated/handlers/index.ts'
import { toFastifySchema } from '../../lib/openapi.ts'
import { toContactResource } from '../../resources/contacts.ts'
import {
  ContactAlreadyExistsError,
  ContactCreateFailedError,
  ContactNotFoundError,
} from '../../use-cases/contacts.errors.ts'
import { createContact, updateContact } from '../../use-cases/contacts.ts'

const contactsPlugin: FastifyPluginAsyncTypebox = async (
  fastify: FastifyInstance,
  _opts,
): Promise<void> => {
  const handlers = {
    contactsList: async (request, reply) => {
      const whereClause = request.query?.status
        ? eq(contacts.status, request.query.status)
        : undefined

      const response: ContactPage = await request.paginate({
        table: contacts,
        query: (table) => fastify.db.select().from(table).where(whereClause).orderBy(asc(table.id)),
        map: toContactResource,
      })

      return reply.code(200).send(response)
    },

    contactsCreate: async (request, reply) => {
      const result = await createContact(fastify.db, request.body)

      if (result.isErr()) {
        if (ContactAlreadyExistsError.is(result.error)) {
          throw fastify.httpErrors.conflict(result.error.message)
        }

        if (ContactCreateFailedError.is(result.error)) {
          throw fastify.httpErrors.internalServerError(result.error.message)
        }

        throw result.error
      }

      return reply.code(201).send(toContactResource(result.value))
    },

    contactsGet: async (request, reply) => {
      const [contact] = await fastify.db
        .select()
        .from(contacts)
        .where(eq(contacts.id, request.params.id))
        .limit(1)

      fastify.guard.found(contact)

      return reply.code(200).send(toContactResource(contact))
    },

    contactsUpdate: async (request, reply) => {
      const result = await updateContact(fastify.db, request.params.id, request.body)

      if (result.isErr()) {
        if (ContactNotFoundError.is(result.error)) {
          throw fastify.httpErrors.notFound(result.error.message)
        }

        throw result.error
      }

      return reply.code(200).send(toContactResource(result.value))
    },

    contactsDelete: async (request, reply) => {
      const [deleted] = await fastify.db
        .delete(contacts)
        .where(eq(contacts.id, request.params.id))
        .returning()

      fastify.guard.found(deleted)

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

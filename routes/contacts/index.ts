import type { FastifyPluginAsyncTypebox } from '@fastify/type-provider-typebox'
import { asc, eq, sql } from 'drizzle-orm'

import { contacts } from '../../db/schema.ts'
import { schema as openapiSchema } from '../../generated/openapi-box.js'
import { buildResponseSchema, type OpenapiOperation } from '../../lib/openapi.ts'
import { toTimestamp } from '../../lib/utils.ts'
import type { RouteHandlers } from '../../types/handlers/fastify.gen.js'
import type { ContactPage, ContactResource } from '../../types/handlers/types.gen.js'

const contactsListOperation = openapiSchema['/contacts'].GET as OpenapiOperation
const contactsCreateOperation = openapiSchema['/contacts'].POST as OpenapiOperation
const contactsGetOperation = openapiSchema['/contacts/{id}'].GET as OpenapiOperation
const contactsUpdateOperation = openapiSchema['/contacts/{id}'].PUT as OpenapiOperation
const contactsDeleteOperation = openapiSchema['/contacts/{id}'].DELETE as OpenapiOperation

const contactsListQuerySchema = contactsListOperation.args.properties.query
const contactsCreateBodySchema = contactsCreateOperation.args.properties.body
const contactParamsSchema = contactsGetOperation.args.properties.params
const contactsUpdateBodySchema = contactsUpdateOperation.args.properties.body

const toContactResource = (row: typeof contacts.$inferSelect): ContactResource => {
  const resource: ContactResource = {
    id: row.id,
    email: row.email,
    status: row.status,
    createdAt: toTimestamp(row.createdAt),
    updatedAt: toTimestamp(row.updatedAt),
  }

  if (row.firstName !== null) {
    resource.firstName = row.firstName
  }

  if (row.lastName !== null) {
    resource.lastName = row.lastName
  }

  if (row.timeZone !== null) {
    resource.timeZone = row.timeZone
  }

  if (row.customFields !== null) {
    resource.customFields = row.customFields
  }

  return resource
}

const contactsPlugin: FastifyPluginAsyncTypebox = async (fastify, _opts): Promise<void> => {
  const handlers = {
    contactsList: async (request, reply) => {
      const page = request.query?.page ?? 0
      const size = request.query?.size ?? 25
      const whereClause = request.query?.status
        ? eq(contacts.status, request.query.status)
        : undefined

      const countRows = await fastify.db
        .select({ totalElements: sql<number>`count(*)` })
        .from(contacts)
        .where(whereClause)
      const totalElements = countRows[0]?.totalElements ?? 0

      const rows = await fastify.db
        .select()
        .from(contacts)
        .where(whereClause)
        .orderBy(asc(contacts.id))
        .limit(size)
        .offset(page * size)

      const response: ContactPage = {
        content: rows.map(toContactResource),
        pageable: {
          pageSize: size,
          pageNumber: page,
          offset: page * size,
          paged: true,
          unpaged: false,
          sort: {
            sorted: true,
            unsorted: false,
            empty: false,
          },
        },
        totalElements,
        totalPages: totalElements === 0 ? 0 : Math.ceil(totalElements / size),
        size,
        number: page,
        numberOfElements: rows.length,
        first: page === 0,
        last: page * size + rows.length >= totalElements,
        empty: rows.length === 0,
        sort: {
          sorted: true,
          unsorted: false,
          empty: false,
        },
      }

      return reply.code(200).send(response)
    },

    contactsCreate: async (request, reply) => {
      try {
        const [created] = await fastify.db
          .insert(contacts)
          .values({
            email: request.body.email,
            firstName: request.body.firstName ?? null,
            lastName: request.body.lastName ?? null,
            status: 'active',
            timeZone: request.body.timeZone ?? null,
            customFields: request.body.customFields ?? null,
            updatedAt: new Date(),
          })
          .returning()

        if (!created) {
          throw new Error('Failed to create contact')
        }

        return reply.code(201).send(toContactResource(created))
      } catch (error) {
        const dbError = error as { code?: string }

        if (dbError.code === '23505') {
          throw fastify.httpErrors.conflict('Contact with this email already exists')
        }

        throw error
      }
    },

    contactsGet: async (request, reply) => {
      const [contact] = await fastify.db
        .select()
        .from(contacts)
        .where(eq(contacts.id, request.params.id))
        .limit(1)

      if (!contact) {
        throw fastify.httpErrors.notFound(`Contact with id=${request.params.id} was not found`)
      }

      return reply.code(200).send(toContactResource(contact))
    },

    contactsUpdate: async (request, reply) => {
      const updates: Partial<typeof contacts.$inferInsert> = {
        updatedAt: new Date(),
      }

      if ('firstName' in request.body) {
        updates.firstName = request.body.firstName ?? null
      }

      if ('lastName' in request.body) {
        updates.lastName = request.body.lastName ?? null
      }

      if ('timeZone' in request.body) {
        updates.timeZone = request.body.timeZone ?? null
      }

      if ('customFields' in request.body) {
        updates.customFields = request.body.customFields ?? null
      }

      try {
        const [updated] = await fastify.db
          .update(contacts)
          .set(updates)
          .where(eq(contacts.id, request.params.id))
          .returning()

        if (!updated) {
          throw fastify.httpErrors.notFound(`Contact with id=${request.params.id} was not found`)
        }

        return reply.code(200).send(toContactResource(updated))
      } catch (error) {
        const dbError = error as { code?: string }

        if (dbError.code === '23505') {
          throw fastify.httpErrors.conflict('Contact with this email already exists')
        }

        throw error
      }
    },

    contactsDelete: async (request, reply) => {
      const [deleted] = await fastify.db
        .delete(contacts)
        .where(eq(contacts.id, request.params.id))
        .returning({ id: contacts.id })

      if (!deleted) {
        throw fastify.httpErrors.notFound(`Contact with id=${request.params.id} was not found`)
      }

      return reply.code(204).send()
    },
  } satisfies Pick<
    RouteHandlers,
    'contactsList' | 'contactsCreate' | 'contactsGet' | 'contactsUpdate' | 'contactsDelete'
  >

  fastify.get(
    '/contacts',
    {
      schema: {
        querystring: contactsListQuerySchema,
        response: buildResponseSchema(contactsListOperation),
      },
    },
    handlers.contactsList,
  )
  fastify.post(
    '/contacts',
    {
      schema: {
        body: contactsCreateBodySchema,
        response: buildResponseSchema(contactsCreateOperation),
      },
    },
    handlers.contactsCreate,
  )
  fastify.get(
    '/contacts/:id',
    {
      schema: { params: contactParamsSchema, response: buildResponseSchema(contactsGetOperation) },
    },
    handlers.contactsGet,
  )
  fastify.put(
    '/contacts/:id',
    {
      schema: {
        params: contactParamsSchema,
        body: contactsUpdateBodySchema,
        response: buildResponseSchema(contactsUpdateOperation),
      },
    },
    handlers.contactsUpdate,
  )
  fastify.delete(
    '/contacts/:id',
    {
      schema: {
        params: contactParamsSchema,
        response: buildResponseSchema(contactsDeleteOperation),
      },
    },
    handlers.contactsDelete,
  )
}

export default contactsPlugin

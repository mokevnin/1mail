import type { FastifyInstance, FastifyReply, FastifyRequest } from 'fastify'

export class Handlers {
  fastify: FastifyInstance

  constructor(fastify: FastifyInstance) {
    this.fastify = fastify
  }

  // Broadcasts
  async Broadcasts_create(_req: FastifyRequest, reply: FastifyReply) {
    return reply.status(201).send({ id: 1, name: 'Sample' })
  }

  async Broadcasts_list(_req: FastifyRequest, _reply: FastifyReply) {
    return {
      content: [],
      totalElements: 0,
      totalPages: 0,
      size: 25,
      number: 0,
      numberOfElements: 0,
      first: true,
      last: true,
      empty: true,
    }
  }

  async Broadcasts_get(_req: FastifyRequest, _reply: FastifyReply) {
    return { id: 1, name: 'Sample' }
  }

  async Broadcasts_update(_req: FastifyRequest, _reply: FastifyReply) {
    return { id: 1, name: 'Updated' }
  }

  async Broadcasts_delete(_req: FastifyRequest, reply: FastifyReply) {
    return reply.status(204).send()
  }

  // Contacts
  async Contacts_create(_req: FastifyRequest, reply: FastifyReply) {
    return reply.status(201).send({ id: 1, email: 'test@example.com' })
  }

  async Contacts_list(_req: FastifyRequest, _reply: FastifyReply) {
    return {
      content: [],
      totalElements: 0,
      totalPages: 0,
      size: 25,
      number: 0,
      numberOfElements: 0,
      first: true,
      last: true,
      empty: true,
    }
  }

  async Contacts_get(_req: FastifyRequest, _reply: FastifyReply) {
    return { id: 1, email: 'test@example.com' }
  }

  async Contacts_update(_req: FastifyRequest, _reply: FastifyReply) {
    return { id: 1, email: 'updated@example.com' }
  }

  async Contacts_delete(_req: FastifyRequest, reply: FastifyReply) {
    return reply.status(204).send()
  }

  // Segments
  async Segments_create(_req: FastifyRequest, reply: FastifyReply) {
    return reply.status(201).send({ id: 1, name: 'Segment' })
  }

  async Segments_list(_req: FastifyRequest, _reply: FastifyReply) {
    return {
      content: [],
      totalElements: 0,
      totalPages: 0,
      size: 25,
      number: 0,
      numberOfElements: 0,
      first: true,
      last: true,
      empty: true,
    }
  }

  async Segments_get(_req: FastifyRequest, _reply: FastifyReply) {
    return { id: 1, name: 'Segment' }
  }

  async Segments_update(_req: FastifyRequest, _reply: FastifyReply) {
    return { id: 1, name: 'Updated Segment' }
  }

  async Segments_delete(_req: FastifyRequest, reply: FastifyReply) {
    return reply.status(204).send()
  }
}

export default (fastify: FastifyInstance) => new Handlers(fastify)

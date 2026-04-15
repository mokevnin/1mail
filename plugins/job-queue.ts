import { PgStorage, Queue } from '@platformatic/job-queue'
import fp from 'fastify-plugin'

declare module 'fastify' {
  interface FastifyInstance {
    queue: Queue<unknown, unknown>
  }
}

export default fp(async (fastify) => {
  const connectionString =
    process.env.DATABASE_URL || 'postgres://postgres:postgres@localhost:5432/1mail'

  const storage = new PgStorage({
    connectionString,
    tablePrefix: 'job_queue_',
  })

  const queue = new Queue({
    storage,
    concurrency: 5,
  })

  // Sample job handler
  queue.execute(async (job) => {
    fastify.log.info({ jobId: job.id, payload: job.payload }, 'Processing job')
    // Simulate work
    await new Promise((resolve) => setTimeout(resolve, 1000))
    return { success: true }
  })

  fastify.decorate('queue', queue)

  fastify.addHook('onReady', async () => {
    await queue.start()
    fastify.log.info('Job queue started')
  })

  fastify.addHook('onClose', async () => {
    await queue.stop()
    fastify.log.info('Job queue stopped')
  })
})

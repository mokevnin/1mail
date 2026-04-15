import { PgStorage, Queue } from '@platformatic/job-queue'
import fp from 'fastify-plugin'

type JobPayload = Record<string, unknown>
type JobResult = { success: true }
type QueueJob = Parameters<Parameters<Queue<JobPayload, JobResult>['execute']>[0]>[0]

export default fp(async (fastify) => {
  const connectionString = process.env.DATABASE_URL
  if (!connectionString) {
    throw new Error('DATABASE_URL is required')
  }

  const storage = new PgStorage({
    connectionString,
    tablePrefix: 'job_queue_',
  })

  const queue = new Queue<JobPayload, JobResult>({
    storage,
    concurrency: 5,
  })

  // Sample job handler
  queue.execute(async (job: QueueJob) => {
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

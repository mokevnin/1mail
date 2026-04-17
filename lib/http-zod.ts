import { z } from 'zod'
import {
  zCollectEventInput,
  zCollectEventsInput,
  zContactStatus,
  zCreateApiTokenInput,
  zEventInput,
  zPageQueryPage,
  zPageQueryPageSize,
  zRecordEventsInput,
} from '../generated/handlers/zod.gen.ts'

const coerceStringNumber = <TSchema extends z.ZodType>(schema: TSchema) =>
  z.preprocess((value) => (typeof value === 'string' ? Number(value) : value), schema)

const zDateTimeInput = z.iso.datetime().transform((value) => new Date(value))

export const zPageQueryInput = z.object({
  page: coerceStringNumber(zPageQueryPage),
  pageSize: coerceStringNumber(zPageQueryPageSize),
})

export const zContactsListQueryInput = zPageQueryInput.extend({
  status: zContactStatus.optional(),
})

export const zCreateApiTokenBodyInput = zCreateApiTokenInput.extend({
  expiresAt: zDateTimeInput.nullish(),
})

const zEventBodyInput = zEventInput.extend({
  occurredAt: zDateTimeInput.nullish(),
})

export const zRecordEventsBodyInput = zRecordEventsInput.extend({
  events: z.array(zEventBodyInput),
})

const zCollectEventBodyInput = zCollectEventInput.extend({
  occurredAt: zDateTimeInput.nullish(),
})

export const zCollectEventsBodyInput = zCollectEventsInput.extend({
  events: z.array(zCollectEventBodyInput),
})

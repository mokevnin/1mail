import { z } from 'zod'
import {
  zCollectEventInput,
  zCollectEventsInput,
  zContactStatus,
  zContactsCreateBody,
  zContactsUpdateBody,
  zCreateApiTokenInput,
  zEmailAddress,
  zEventInput,
  zPageQueryPage,
  zPageQueryPageSize,
  zRecordEventsInput,
} from '../generated/handlers/zod.gen.ts'
import { zSiteContactsCreateBody, zSiteContactsUpdateBody } from '../generated/site/zod.gen.ts'

const coerceStringNumber = <TSchema extends z.ZodType>(schema: TSchema) =>
  z.preprocess((value) => (typeof value === 'string' ? Number(value) : value), schema)

const trimString = (value: unknown) => (typeof value === 'string' ? value.trim() : value)

export const zTrimmedEmailInput = z.preprocess(trimString, zEmailAddress)

export const zNullableTextInput = z.preprocess((value) => {
  if (typeof value !== 'string') {
    return value
  }

  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}, z.string().nullish())

const zDateTimeInput = z.iso.datetime().transform((value) => new Date(value))

export const zPageQueryInput = z.object({
  page: coerceStringNumber(zPageQueryPage),
  pageSize: coerceStringNumber(zPageQueryPageSize),
})

export const zContactsListQueryInput = zPageQueryInput.extend({
  status: zContactStatus.optional(),
})

export const zContactsCreateBodyInput = zContactsCreateBody.extend({
  email: zTrimmedEmailInput,
  firstName: zNullableTextInput,
  lastName: zNullableTextInput,
  timeZone: zNullableTextInput,
})

export const zContactsUpdateBodyInput = zContactsUpdateBody.extend({
  firstName: zNullableTextInput,
  lastName: zNullableTextInput,
  timeZone: zNullableTextInput,
})

export const zSiteContactsCreateBodyInput = zSiteContactsCreateBody.extend({
  email: zTrimmedEmailInput,
  firstName: zNullableTextInput,
  lastName: zNullableTextInput,
  timeZone: zNullableTextInput,
})

export const zSiteContactsUpdateBodyInput = zSiteContactsUpdateBody.extend({
  firstName: zNullableTextInput,
  lastName: zNullableTextInput,
  timeZone: zNullableTextInput,
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

import * as z from 'zod'

// Form state is raw strings, keyed by the edited field names.
export type FormValues = Record<string, string>

// What the form layer hands to the API schema: a string, or the API's "no value" markers.
type Wire = Record<string, string | null | undefined>

// Every edited field must be a string field on the API side.
type StringFieldsSchema = z.ZodObject<Record<string, z.ZodType<unknown, string | null | undefined>>>

// The generated payload types declare optional keys without `| undefined`
// (exactOptionalPropertyTypes), while zod's output type adds it. The codec already drops
// absent fields at runtime, so only the type needs narrowing.
type Exact<T> = { [K in keyof T]: Exclude<T[K], undefined> }

function isExact<T extends object>(value: object): value is Exact<T> {
  return Object.values(value).every((entry) => entry !== undefined)
}

function exact<T extends object>(value: T): Exact<T> {
  if (!isExact<T>(value)) throw new Error('unreachable: the codec drops absent fields')
  return value
}

// The one place form strings meet the API's null/absent semantics (JSON Merge Patch: absent
// keeps a value, null clears it), as a zod codec between the two.
//   - decode (form -> wire): trim; a blank value is `blankOf(field)`, or dropped when undefined.
//   - encode (wire -> form): null or absent becomes a blank string, so inputs never see null.
function formCodec(blankOf: (field: string) => string | null | undefined) {
  return z.codec(z.record(z.string(), z.string()), z.record(z.string(), z.string().nullish()), {
    decode: (values) =>
      Object.fromEntries(
        Object.entries(values).flatMap(([field, value]) => {
          const decoded = value.trim() || blankOf(field)
          return decoded === undefined ? [] : [[field, decoded]]
        }),
      ),
    encode: (wire: Wire) =>
      Object.fromEntries(Object.entries(wire).map(([field, value]) => [field, value ?? ''])),
  })
}

// A bidirectional form schema derived from a generated (TypeSpec) zod object schema. The
// codec is piped into the generated schema, so TypeSpec stays the single source of truth for
// validation.
//   - create: a blank field is absent; a blank required field is reported missing by the schema.
//   - update: a blank field is cleared: null where the API field is nullable, an empty string
//     for other optional fields; a blank required field is reported missing.
//   - toValues: loaded resource -> form values.
// Pick the edited fields from the generated schema (`zSiteUpdateContactInput.pick({ email: true })`);
// anything not picked never round-trips into a payload.
// What a loaded resource must offer: each edited field as a string, null or absent.
export type ResourceValues<TPayload> = { [K in keyof TPayload]?: string | null | undefined }

export type ResourceFormSchema<TPayload> = {
  // Form values with every field blank.
  blank: FormValues
  create: z.ZodType<TPayload, FormValues>
  update: z.ZodType<TPayload, FormValues>
  toValues: (resource: ResourceValues<TPayload>) => FormValues
}

export function resourceFormSchema<TSchema extends StringFieldsSchema>(
  schema: TSchema,
): ResourceFormSchema<Exact<z.output<TSchema>>> {
  const fields = Object.keys(schema.shape)
  // optional/nullable come from zod's own isOptional()/isNullable() on each field schema.
  const clearedOnUpdate = (field: string) => {
    const shape = schema.shape[field]
    if (!shape?.isOptional()) return undefined
    return shape.isNullable() ? null : ''
  }
  const decoder = formCodec(clearedOnUpdate)
  const form = (codec: typeof decoder) => codec.pipe(schema).transform(exact)
  return {
    blank: Object.fromEntries(fields.map((field) => [field, ''])),
    create: form(formCodec(() => undefined)),
    update: form(decoder),
    toValues: (resource) => {
      const values = new Map<string, unknown>(Object.entries(resource))
      return z.encode(
        decoder,
        Object.fromEntries(
          fields.map((field) => {
            const value = values.get(field)
            return [field, typeof value === 'string' ? value : null]
          }),
        ),
      )
    },
  }
}

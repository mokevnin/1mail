import * as z from 'zod'

// Form state is raw strings, keyed by the edited field names.
export type FormValues = Record<string, string>

// The generated payload types declare optional keys without `| undefined`
// (exactOptionalPropertyTypes), while zod's output type adds it.
type Exact<T> = { [K in keyof T]: Exclude<T[K], undefined> }

// A bidirectional form schema derived from a generated (TypeSpec) zod object schema.
//   - create / update (form -> payload): trim, then a blank string becomes "absent" on create
//     and an explicit null on update (an empty string for fields that are not nullable) (JSON Merge Patch: absent keeps a value, null clears it);
//     the result is validated by the generated schema, so TypeSpec stays the single source of
//     truth for validation.
//   - toValues (resource -> form): null or absent becomes a blank string, so inputs never see null.
// The form edits exactly the fields of the schema passed in: pick them from the generated
// schema (`zSiteUpdateContactInput.pick({ email: true })`); anything not picked (e.g. contact
// custom fields) never round-trips into a payload.
export type ResourceFormSchema<TPayload> = {
  // Form values with every field blank.
  blank: FormValues
  // Validates form values and produces the create payload (blank fields omitted).
  create: z.ZodType<TPayload, FormValues>
  // Validates form values and produces the update payload (blank fields sent as null).
  update: z.ZodType<TPayload, FormValues>
  // Maps a loaded resource to form values.
  toValues: (resource: object) => FormValues
}

type Mode = 'create' | 'update'

// A blank field on update clears it: null where the API field is nullable, otherwise an empty string.
function normalize(values: FormValues, mode: Mode, nullable: ReadonlySet<string>) {
  const normalized: Record<string, string | null | undefined> = {}
  for (const [field, value] of Object.entries(values)) {
    const trimmed = value.trim()
    if (trimmed !== '') normalized[field] = trimmed
    else if (mode === 'update') normalized[field] = nullable.has(field) ? null : ''
  }
  return normalized
}

// Sound guard: Exact<T> differs from T only by the absence of undefined values.
function isExact<T extends object>(value: object): value is Exact<T> {
  return Object.values(value).every((entry) => entry !== undefined)
}

function omitUndefined<T extends object>(value: T): Exact<T> {
  const entries = Object.entries(value).filter(([, entry]) => entry !== undefined)
  const result = Object.fromEntries(entries)
  if (!isExact<T>(result)) throw new Error('unreachable: undefined values were filtered out')
  return result
}

function toFormValue(value: unknown) {
  return typeof value === 'string' ? value : ''
}

// Every picked field must be a string field on the API side.
type StringFieldsSchema = z.ZodObject<Record<string, z.ZodType<unknown, string | null | undefined>>>

export function resourceFormSchema<TSchema extends StringFieldsSchema>(
  schema: TSchema,
): ResourceFormSchema<Exact<z.output<TSchema>>> {
  const fields = Object.keys(schema.shape)
  const nullable = new Set(fields.filter((field) => schema.shape[field]?.safeParse(null).success))
  const parse = (mode: Mode) =>
    z
      .record(z.string(), z.string())
      .transform((values) => normalize(values, mode, nullable))
      .pipe(schema)
      .transform((payload) => omitUndefined(payload))
  return {
    blank: Object.fromEntries(fields.map((field) => [field, ''])),
    create: parse('create'),
    update: parse('update'),
    toValues: (resource) =>
      Object.fromEntries(fields.map((field) => [field, toFormValue(Reflect.get(resource, field))])),
  }
}

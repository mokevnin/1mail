/* eslint eslint-comments/no-unlimited-disable: off */
/* eslint-disable */
// This document was generated automatically by openapi-box

/**
 * @typedef {import('@sinclair/typebox').TSchema} TSchema
 */

/**
 * @template {TSchema} T
 * @typedef {import('@sinclair/typebox').Static<T>} Static
 */

/**
 * @typedef {import('@sinclair/typebox').SchemaOptions} SchemaOptions
 */

/**
 * @typedef {{
 *  [Path in keyof typeof schema]: {
 *    [Method in keyof typeof schema[Path]]: {
 *      [Prop in keyof typeof schema[Path][Method]]: typeof schema[Path][Method][Prop] extends TSchema ?
 *        Static<typeof schema[Path][Method][Prop]> :
 *        undefined
 *    }
 *  }
 * }} SchemaType
 */

/**
 * @typedef {{
 *  [ComponentType in keyof typeof _components]: {
 *    [ComponentName in keyof typeof _components[ComponentType]]: typeof _components[ComponentType][ComponentName] extends TSchema ?
 *      Static<typeof _components[ComponentType][ComponentName]> :
 *      undefined
 *  }
 * }} ComponentType
 */

import { Type as T, TypeRegistry, Kind, CloneType } from '@sinclair/typebox'
import { Value } from '@sinclair/typebox/value'

/**
 * @typedef {{
 *  [Kind]: 'Binary'
 *  static: string | File | Blob | Uint8Array
 *  anyOf: [{
 *    type: 'object',
 *    additionalProperties: true
 *  }, {
 *    type: 'string',
 *    format: 'binary'
 *  }]
 * } & TSchema} TBinary
 */

/**
 * @returns {TBinary}
 */
const Binary = () => {
  /**
   * @param {TBinary} schema
   * @param {unknown} value
   * @returns {boolean}
   */
  function BinaryCheck(schema, value) {
    const type = Object.prototype.toString.call(value)
    return (
      type === '[object Blob]' ||
      type === '[object File]' ||
      type === '[object String]' ||
      type === '[object Uint8Array]'
    )
  }

  if (!TypeRegistry.Has('Binary')) TypeRegistry.Set('Binary', BinaryCheck)

  return /** @type {TBinary} */ ({
    anyOf: [
      {
        type: 'object',
        additionalProperties: true
      },
      {
        type: 'string',
        format: 'binary'
      }
    ],
    [Kind]: 'Binary'
  })
}

const ComponentsSchemasBroadcastStatus = T.Union([
  T.Literal('draft'),
  T.Literal('scheduled'),
  T.Literal('sent')
])
const ComponentsSchemasTimestamp = T.String({ format: 'date-time' })
const ComponentsSchemasBroadcastResource = T.Object({
  id: T.Integer({ format: 'int64' }),
  name: T.String(),
  segmentId: T.Integer({ format: 'int64' }),
  status: T.Intersect([CloneType(ComponentsSchemasBroadcastStatus)]),
  createdAt: T.Intersect([CloneType(ComponentsSchemasTimestamp)]),
  updatedAt: T.Intersect([CloneType(ComponentsSchemasTimestamp)])
})
const ComponentsSchemasProblemDetails = T.Object({
  type: T.Optional(T.String()),
  title: T.Optional(T.String()),
  status: T.Optional(T.Integer({ format: 'int32' })),
  detail: T.Optional(T.String()),
  instance: T.Optional(T.String()),
  errors: T.Optional(
    T.Object(
      {},
      {
        additionalProperties: T.Array(T.String())
      }
    )
  )
})
const ComponentsSchemasCreateBroadcastInput = T.Object({
  name: T.String(),
  segmentId: T.Integer({ format: 'int64' }),
  status: T.Intersect([CloneType(ComponentsSchemasBroadcastStatus)])
})
const ComponentsParametersPageQueryPage = T.Any()
const ComponentsParametersPageQuerySize = T.Any()
const ComponentsParametersPageQuerySort = T.Any()
const ComponentsSchemasSort = T.Object({
  sorted: T.Optional(T.Boolean()),
  unsorted: T.Optional(T.Boolean()),
  empty: T.Optional(T.Boolean())
})
const ComponentsSchemasPageable = T.Object({
  pageSize: T.Optional(T.Integer({ format: 'int32' })),
  pageNumber: T.Optional(T.Integer({ format: 'int32' })),
  offset: T.Optional(T.Integer({ format: 'int64' })),
  paged: T.Optional(T.Boolean()),
  unpaged: T.Optional(T.Boolean()),
  sort: T.Optional(T.Intersect([CloneType(ComponentsSchemasSort)]))
})
const ComponentsSchemasBroadcastPage = T.Object({
  content: T.Array(CloneType(ComponentsSchemasBroadcastResource)),
  pageable: T.Optional(T.Intersect([CloneType(ComponentsSchemasPageable)])),
  totalElements: T.Integer({ format: 'int64' }),
  totalPages: T.Integer({ format: 'int32' }),
  size: T.Integer({ format: 'int32' }),
  number: T.Integer({ format: 'int32' }),
  numberOfElements: T.Integer({ format: 'int32' }),
  first: T.Boolean(),
  last: T.Boolean(),
  empty: T.Boolean(),
  sort: T.Optional(T.Intersect([CloneType(ComponentsSchemasSort)]))
})
const ComponentsSchemasUpdateBroadcastInput = T.Object({
  name: T.Optional(T.Union([T.Null(), T.String()])),
  segmentId: T.Optional(T.Union([T.Null(), T.Integer({ format: 'int64' })])),
  status: T.Optional(T.Intersect([CloneType(ComponentsSchemasBroadcastStatus)]))
})
const ComponentsSchemasEmailAddress = T.String({ format: 'email' })
const ComponentsSchemasContactStatus = T.Union([
  T.Literal('active'),
  T.Literal('unsubscribed')
])
const ComponentsSchemasTimeZoneName = T.String()
const ComponentsSchemasContactResource = T.Object({
  id: T.Integer({ format: 'int64' }),
  email: T.Intersect([CloneType(ComponentsSchemasEmailAddress)]),
  firstName: T.Optional(T.String()),
  lastName: T.Optional(T.String()),
  status: T.Intersect([CloneType(ComponentsSchemasContactStatus)]),
  timeZone: T.Optional(T.Intersect([CloneType(ComponentsSchemasTimeZoneName)])),
  customFields: T.Optional(
    T.Object(
      {},
      {
        additionalProperties: T.String()
      }
    )
  ),
  createdAt: T.Intersect([CloneType(ComponentsSchemasTimestamp)]),
  updatedAt: T.Intersect([CloneType(ComponentsSchemasTimestamp)])
})
const ComponentsSchemasCreateContactInput = T.Object({
  email: T.Intersect([CloneType(ComponentsSchemasEmailAddress)]),
  firstName: T.Optional(T.String()),
  lastName: T.Optional(T.String()),
  timeZone: T.Optional(T.Intersect([CloneType(ComponentsSchemasTimeZoneName)])),
  customFields: T.Optional(
    T.Object(
      {},
      {
        additionalProperties: T.String()
      }
    )
  )
})
const ComponentsSchemasContactPage = T.Object({
  content: T.Array(CloneType(ComponentsSchemasContactResource)),
  pageable: T.Optional(T.Intersect([CloneType(ComponentsSchemasPageable)])),
  totalElements: T.Integer({ format: 'int64' }),
  totalPages: T.Integer({ format: 'int32' }),
  size: T.Integer({ format: 'int32' }),
  number: T.Integer({ format: 'int32' }),
  numberOfElements: T.Integer({ format: 'int32' }),
  first: T.Boolean(),
  last: T.Boolean(),
  empty: T.Boolean(),
  sort: T.Optional(T.Intersect([CloneType(ComponentsSchemasSort)]))
})
const ComponentsSchemasUpdateContactInput = T.Object({
  firstName: T.Optional(T.Union([T.Null(), T.String()])),
  lastName: T.Optional(T.Union([T.Null(), T.String()])),
  timeZone: T.Optional(
    T.Union([T.Null(), T.Intersect([CloneType(ComponentsSchemasTimeZoneName)])])
  ),
  customFields: T.Optional(
    T.Union([
      T.Null(),
      T.Object(
        {},
        {
          additionalProperties: T.String()
        }
      )
    ])
  )
})
const ComponentsSchemasSegmentType = T.Union([
  T.Literal('rule'),
  T.Literal('snapshot')
])
const ComponentsSchemasSegmentResource = T.Object({
  id: T.Integer({ format: 'int64' }),
  name: T.String(),
  type: T.Intersect([CloneType(ComponentsSchemasSegmentType)]),
  definition: T.Optional(T.String()),
  createdAt: T.Intersect([CloneType(ComponentsSchemasTimestamp)]),
  updatedAt: T.Intersect([CloneType(ComponentsSchemasTimestamp)])
})
const ComponentsSchemasCreateSegmentInput = T.Object({
  name: T.String(),
  type: T.Intersect([CloneType(ComponentsSchemasSegmentType)]),
  definition: T.Optional(T.String())
})
const ComponentsSchemasSegmentPage = T.Object({
  content: T.Array(CloneType(ComponentsSchemasSegmentResource)),
  pageable: T.Optional(T.Intersect([CloneType(ComponentsSchemasPageable)])),
  totalElements: T.Integer({ format: 'int64' }),
  totalPages: T.Integer({ format: 'int32' }),
  size: T.Integer({ format: 'int32' }),
  number: T.Integer({ format: 'int32' }),
  numberOfElements: T.Integer({ format: 'int32' }),
  first: T.Boolean(),
  last: T.Boolean(),
  empty: T.Boolean(),
  sort: T.Optional(T.Intersect([CloneType(ComponentsSchemasSort)]))
})
const ComponentsSchemasUpdateSegmentInput = T.Object({
  name: T.Optional(T.Union([T.Null(), T.String()])),
  type: T.Optional(T.Intersect([CloneType(ComponentsSchemasSegmentType)])),
  definition: T.Optional(T.Union([T.Null(), T.String()]))
})

const schema = {
  '/broadcasts': {
    POST: {
      args: T.Object({
        body: CloneType(ComponentsSchemasCreateBroadcastInput, {
          'x-content-type': 'application/json'
        })
      }),
      data: CloneType(ComponentsSchemasBroadcastResource, {
        'x-status-code': '201',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '422',
          'x-content-type': 'application/problem+json'
        })
      ])
    },
    GET: {
      args: T.Optional(
        T.Object({
          query: T.Optional(
            T.Object({
              page: T.Optional(
                T.Integer({ format: 'int32', default: 0, 'x-in': 'query' })
              ),
              size: T.Optional(
                T.Integer({ format: 'int32', default: 25, 'x-in': 'query' })
              ),
              sort: T.Optional(T.Array(T.String(), { 'x-in': 'query' }))
            })
          )
        })
      ),
      data: CloneType(ComponentsSchemasBroadcastPage, {
        'x-status-code': '200',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '422',
          'x-content-type': 'application/problem+json'
        })
      ])
    }
  },
  '/broadcasts/{id}': {
    GET: {
      args: T.Object({
        params: T.Object({
          id: T.Integer({ format: 'int64', 'x-in': 'path' })
        })
      }),
      data: CloneType(ComponentsSchemasBroadcastResource, {
        'x-status-code': '200',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '404',
          'x-content-type': 'application/problem+json'
        })
      ])
    },
    PUT: {
      args: T.Object({
        params: T.Object({
          id: T.Integer({ format: 'int64', 'x-in': 'path' })
        }),
        body: CloneType(ComponentsSchemasUpdateBroadcastInput, {
          'x-content-type': 'application/json'
        })
      }),
      data: CloneType(ComponentsSchemasBroadcastResource, {
        'x-status-code': '200',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '404',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '422',
          'x-content-type': 'application/problem+json'
        })
      ])
    },
    DELETE: {
      args: T.Object({
        params: T.Object({
          id: T.Integer({ format: 'int64', 'x-in': 'path' })
        })
      }),
      data: T.Any({ 'x-status-code': '204' }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '404',
          'x-content-type': 'application/problem+json'
        })
      ])
    }
  },
  '/contacts': {
    POST: {
      args: T.Object({
        body: CloneType(ComponentsSchemasCreateContactInput, {
          'x-content-type': 'application/json'
        })
      }),
      data: CloneType(ComponentsSchemasContactResource, {
        'x-status-code': '201',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '409',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '422',
          'x-content-type': 'application/problem+json'
        })
      ])
    },
    GET: {
      args: T.Optional(
        T.Object({
          query: T.Optional(
            T.Object({
              page: T.Optional(
                T.Integer({ format: 'int32', default: 0, 'x-in': 'query' })
              ),
              size: T.Optional(
                T.Integer({ format: 'int32', default: 25, 'x-in': 'query' })
              ),
              sort: T.Optional(T.Array(T.String(), { 'x-in': 'query' })),
              status: T.Optional(
                CloneType(ComponentsSchemasContactStatus, { 'x-in': 'query' })
              )
            })
          )
        })
      ),
      data: CloneType(ComponentsSchemasContactPage, {
        'x-status-code': '200',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '422',
          'x-content-type': 'application/problem+json'
        })
      ])
    }
  },
  '/contacts/{id}': {
    GET: {
      args: T.Object({
        params: T.Object({
          id: T.Integer({ format: 'int64', 'x-in': 'path' })
        })
      }),
      data: CloneType(ComponentsSchemasContactResource, {
        'x-status-code': '200',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '404',
          'x-content-type': 'application/problem+json'
        })
      ])
    },
    PUT: {
      args: T.Object({
        params: T.Object({
          id: T.Integer({ format: 'int64', 'x-in': 'path' })
        }),
        body: CloneType(ComponentsSchemasUpdateContactInput, {
          'x-content-type': 'application/json'
        })
      }),
      data: CloneType(ComponentsSchemasContactResource, {
        'x-status-code': '200',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '404',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '409',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '422',
          'x-content-type': 'application/problem+json'
        })
      ])
    },
    DELETE: {
      args: T.Object({
        params: T.Object({
          id: T.Integer({ format: 'int64', 'x-in': 'path' })
        })
      }),
      data: T.Any({ 'x-status-code': '204' }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '404',
          'x-content-type': 'application/problem+json'
        })
      ])
    }
  },
  '/segments': {
    POST: {
      args: T.Object({
        body: CloneType(ComponentsSchemasCreateSegmentInput, {
          'x-content-type': 'application/json'
        })
      }),
      data: CloneType(ComponentsSchemasSegmentResource, {
        'x-status-code': '201',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '422',
          'x-content-type': 'application/problem+json'
        })
      ])
    },
    GET: {
      args: T.Optional(
        T.Object({
          query: T.Optional(
            T.Object({
              page: T.Optional(
                T.Integer({ format: 'int32', default: 0, 'x-in': 'query' })
              ),
              size: T.Optional(
                T.Integer({ format: 'int32', default: 25, 'x-in': 'query' })
              ),
              sort: T.Optional(T.Array(T.String(), { 'x-in': 'query' }))
            })
          )
        })
      ),
      data: CloneType(ComponentsSchemasSegmentPage, {
        'x-status-code': '200',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '422',
          'x-content-type': 'application/problem+json'
        })
      ])
    }
  },
  '/segments/{id}': {
    GET: {
      args: T.Object({
        params: T.Object({
          id: T.Integer({ format: 'int64', 'x-in': 'path' })
        })
      }),
      data: CloneType(ComponentsSchemasSegmentResource, {
        'x-status-code': '200',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '404',
          'x-content-type': 'application/problem+json'
        })
      ])
    },
    PUT: {
      args: T.Object({
        params: T.Object({
          id: T.Integer({ format: 'int64', 'x-in': 'path' })
        }),
        body: CloneType(ComponentsSchemasUpdateSegmentInput, {
          'x-content-type': 'application/json'
        })
      }),
      data: CloneType(ComponentsSchemasSegmentResource, {
        'x-status-code': '200',
        'x-content-type': 'application/json'
      }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '404',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '422',
          'x-content-type': 'application/problem+json'
        })
      ])
    },
    DELETE: {
      args: T.Object({
        params: T.Object({
          id: T.Integer({ format: 'int64', 'x-in': 'path' })
        })
      }),
      data: T.Any({ 'x-status-code': '204' }),
      error: T.Union([
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '400',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '401',
          'x-content-type': 'application/problem+json'
        }),
        CloneType(ComponentsSchemasProblemDetails, {
          'x-status-code': '404',
          'x-content-type': 'application/problem+json'
        })
      ])
    }
  }
}

const _components = {
  parameters: {
    'PageQuery.page': T.Optional(
      T.Integer({ format: 'int32', default: 0, 'x-in': 'query' })
    ),
    'PageQuery.size': T.Optional(
      T.Integer({ format: 'int32', default: 25, 'x-in': 'query' })
    ),
    'PageQuery.sort': T.Optional(T.Array(T.String(), { 'x-in': 'query' }))
  },
  schemas: {
    BroadcastPage: CloneType(ComponentsSchemasBroadcastPage),
    BroadcastResource: CloneType(ComponentsSchemasBroadcastResource),
    BroadcastStatus: CloneType(ComponentsSchemasBroadcastStatus),
    ContactPage: CloneType(ComponentsSchemasContactPage),
    ContactResource: CloneType(ComponentsSchemasContactResource),
    ContactStatus: CloneType(ComponentsSchemasContactStatus, {
      'x-in': 'query'
    }),
    CreateBroadcastInput: CloneType(ComponentsSchemasCreateBroadcastInput),
    CreateContactInput: CloneType(ComponentsSchemasCreateContactInput),
    CreateSegmentInput: CloneType(ComponentsSchemasCreateSegmentInput),
    EmailAddress: CloneType(ComponentsSchemasEmailAddress),
    Pageable: CloneType(ComponentsSchemasPageable),
    ProblemDetails: CloneType(ComponentsSchemasProblemDetails),
    SegmentPage: CloneType(ComponentsSchemasSegmentPage),
    SegmentResource: CloneType(ComponentsSchemasSegmentResource),
    SegmentType: CloneType(ComponentsSchemasSegmentType),
    Sort: CloneType(ComponentsSchemasSort),
    TimeZoneName: CloneType(ComponentsSchemasTimeZoneName),
    Timestamp: CloneType(ComponentsSchemasTimestamp),
    UpdateBroadcastInput: CloneType(ComponentsSchemasUpdateBroadcastInput),
    UpdateContactInput: CloneType(ComponentsSchemasUpdateContactInput),
    UpdateSegmentInput: CloneType(ComponentsSchemasUpdateSegmentInput),
    Versions: T.Literal('1.0.0')
  }
}

export { schema, _components as components }

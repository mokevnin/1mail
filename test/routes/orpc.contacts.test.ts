import { expect, test } from 'vitest'

import { build } from '../helper.ts'

test('orpc contacts endpoints return expected statuses', async (t) => {
  const app = await build(t)

  const listResponse = await app.inject({
    method: 'POST',
    url: '/orpc/siteContactsList',
    payload: {
      json: {
        query: {
          page: 1,
          pageSize: 10,
        },
      },
    },
  })

  expect(listResponse.statusCode).toBe(200)

  const missingGetResponse = await app.inject({
    method: 'POST',
    url: '/orpc/siteContactsGet',
    payload: {
      json: {
        params: {
          id: '999',
        },
      },
    },
  })

  expect(missingGetResponse.statusCode).toBe(404)

  await app.inject({
    method: 'POST',
    url: '/orpc/siteContactsCreate',
    payload: {
      json: {
        body: {
          email: 'duplicate-orpc@example.com',
        },
      },
    },
  })

  const duplicateResponse = await app.inject({
    method: 'POST',
    url: '/orpc/siteContactsCreate',
    payload: {
      json: {
        body: {
          email: 'duplicate-orpc@example.com',
        },
      },
    },
  })

  expect(duplicateResponse.statusCode).toBe(409)

  const missingUpdateResponse = await app.inject({
    method: 'POST',
    url: '/orpc/siteContactsUpdate',
    payload: {
      json: {
        body: {
          firstName: 'Missing',
        },
        params: {
          id: '999',
        },
      },
    },
  })

  expect(missingUpdateResponse.statusCode).toBe(404)

  const missingDeleteResponse = await app.inject({
    method: 'POST',
    url: '/orpc/siteContactsDelete',
    payload: {
      json: {
        params: {
          id: '999',
        },
      },
    },
  })

  expect(missingDeleteResponse.statusCode).toBe(404)
})

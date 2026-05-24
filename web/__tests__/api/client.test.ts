import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'

import { request } from '../../src/api/client'
import { server } from '../../src/mocks/server'
import { setToken } from '../../src/utils/token'

describe('api client', () => {
  it('sends idempotency keys only for write requests with an explicit key', async () => {
    const seenHeaders: Array<string | null> = []
    setToken('mock-access-token')

    server.use(
      http.post('*/api/v1/idempotency-probe', ({ request }) => {
        seenHeaders.push(request.headers.get('idempotency-key'))
        return HttpResponse.json({ code: 'OK', message: 'ok', request_id: 'req-post', data: null })
      }),
      http.get('*/api/v1/idempotency-probe', ({ request }) => {
        seenHeaders.push(request.headers.get('idempotency-key'))
        return HttpResponse.json({ code: 'OK', message: 'ok', request_id: 'req-get', data: null })
      }),
    )

    await request<null>({
      url: '/idempotency-probe',
      method: 'POST',
      idempotencyKey: 'phase9-idempotency-key',
    })
    await request<null>({
      url: '/idempotency-probe',
      method: 'GET',
      idempotencyKey: 'ignored-for-read',
    })

    expect(seenHeaders).toEqual(['phase9-idempotency-key', null])
  })
})

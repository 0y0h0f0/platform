import { describe, expect, it } from 'vitest'

import type { ApiEnvelope, AuthData } from '../../src/api/types'

describe('auth mock handlers', () => {
  it('returns a mocked login envelope', async () => {
    const response = await fetch('http://localhost/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ account: 'demo_user', password: 'password123' }),
    })

    expect(response.status).toBe(200)

    const envelope = (await response.json()) as ApiEnvelope<AuthData>
    expect(envelope.code).toBe('OK')
    expect(envelope.data?.access_token).toBe('mock-access-token')
    expect(envelope.data?.user.username).toBe('demo_user')
  })
})

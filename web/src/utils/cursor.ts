type CursorValue = Record<string, unknown>

function bytesToBase64Url(bytes: Uint8Array): string {
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return window.btoa(binary).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '')
}

function base64UrlToBytes(value: string): Uint8Array {
  const base64 = value.replaceAll('-', '+').replaceAll('_', '/')
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=')
  const binary = window.atob(padded)
  return Uint8Array.from(binary, (char) => char.charCodeAt(0))
}

// encodeCursor mirrors the backend cursor shape for frontend-only mocks/tests.
export function encodeCursor(value: CursorValue): string {
  return bytesToBase64Url(new TextEncoder().encode(JSON.stringify(value)))
}

// decodeCursor restores cursor payloads produced by encodeCursor.
export function decodeCursor<T extends CursorValue = CursorValue>(cursor: string): T {
  const bytes = base64UrlToBytes(cursor)
  return JSON.parse(new TextDecoder().decode(bytes)) as T
}

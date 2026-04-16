import bcrypt from 'bcryptjs'
import { customAlphabet, nanoid } from 'nanoid'

const TOKEN_PREFIX = 'omtk'
const TOKEN_RE = /^omtk_([a-z0-9]+)_([A-Za-z0-9_-]+)$/
const createPrefix = customAlphabet('0123456789abcdefghijklmnopqrstuvwxyz', 12)

export function createTokenValue(prefix: string, secret: string): string {
  return `${TOKEN_PREFIX}_${prefix}_${secret}`
}

export function generateTokenPrefix(): string {
  return createPrefix()
}

export function generateTokenSecret(): string {
  return nanoid(40)
}

export async function hashTokenSecret(secret: string): Promise<string> {
  return bcrypt.hash(secret, 12)
}

export async function verifyTokenSecret(secret: string, secretHash: string): Promise<boolean> {
  return bcrypt.compare(secret, secretHash)
}

export function parseApiToken(
  value: string | undefined,
): { prefix: string; secret: string } | null {
  if (!value) {
    return null
  }

  const match = value.match(TOKEN_RE)
  if (!match) {
    return null
  }

  const prefix = match[1]
  const secret = match[2]
  if (!prefix || !secret) {
    return null
  }

  return { prefix, secret }
}

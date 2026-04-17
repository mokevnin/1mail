import type { EventActionResource } from '#/generated/handlers/index.ts'

export function toEventActionResource(action: string): EventActionResource {
  return { action }
}

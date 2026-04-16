export type UseCaseErrorCode = 'conflict' | 'not_found' | 'internal'

export type UseCaseError = {
  code: UseCaseErrorCode
  message: string
}

export class EntityNotFoundError extends Error {
  constructor() {
    super('Not Found')
    this.name = 'EntityNotFoundError'
  }
}

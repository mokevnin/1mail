using TypeSpec.Http;

namespace OneMail;

/** Reusable pagination query parameters */
model PageQuery {
  /** Page number (1-based) */
  @query page?: int32 = 1;

  /** Page size */
  @query pageSize?: int32 = 25;
}

/** Paginated response */
model Page<T> {
  /** List of items */
  items: T[];

  /** Page number (1-based) */
  page: int32;

  /** Page size */
  pageSize: int32;

  /** Total number of elements */
  totalItems: int32;

  /** Total number of pages */
  totalPages: int32;
}

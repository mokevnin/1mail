using TypeSpec.Http;

namespace OneMail;

/** Spring Data sort descriptor */
model Sort {
  /** Whether sorting is enabled */
  sorted?: boolean;

  /** Whether sorting is unsorted */
  unsorted?: boolean;

  /** Whether sorting is empty */
  empty?: boolean;
}

/** Spring Data pageable descriptor */
model Pageable {
  /** Page size */
  pageSize?: int32;

  /** Page number (0-based) */
  pageNumber?: int32;

  /** Offset */
  offset?: int64;

  /** Whether the page is paged */
  paged?: boolean;

  /** Whether the page is unpaged */
  unpaged?: boolean;

  /** Sorting information */
  sort?: Sort;
}

/** Reusable pagination query parameters */
model PageQuery {
  /** Page number (0-based) */
  @query page?: int32 = 0;

  /** Page size */
  @query size?: int32 = 25;

  /** Sorting criteria (e.g. name,asc). Can be repeated. */
  @query sort?: string[];
}

/** Spring Data paginated response */
model Page<T> {
  /** List of items */
  content: T[];

  /** Paging information */
  pageable?: Pageable;

  /** Total number of elements */
  totalElements: int64;

  /** Total number of pages */
  totalPages: int32;

  /** Page size */
  size: int32;

  /** Page number (0-based) */
  number: int32;

  /** Number of elements in current page */
  numberOfElements: int32;

  /** Whether this is the first page */
  first: boolean;

  /** Whether this is the last page */
  last: boolean;

  /** Whether content is empty */
  empty: boolean;

  /** Sorting information */
  sort?: Sort;
}

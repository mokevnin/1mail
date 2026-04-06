import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

@service(#{ title: "OneMail API" })
@info(#{ version: "1.0.0" })
@route("/api")
@useAuth(BearerAuth)
namespace OneMail;

// ─── Models ─────────────────────────────────────────────────────────────────

/** Contact status */
enum ContactStatus {
  active,
  unsubscribed,
}

/** Contact response resource */
@example(#{
  id: 101,
  email: "alice@example.com",
  firstName: "Alice",
  lastName: "Johnson",
  status: ContactStatus.active,
  timeZone: "America/New_York",
  customFields: #{
    plan: "pro",
    source: "import",
  },
  createdAt: "2026-04-01T10:15:30Z",
  updatedAt: "2026-04-03T08:45:00Z",
})
model ContactResponse {
  /** Unique identifier */
  id: int64;

  /** Email address */
  email: string;

  /** First name */
  firstName?: string;

  /** Last name */
  lastName?: string;

  /** Current status */
  status: ContactStatus;

  /** IANA time zone identifier */
  timeZone?: string;

  /** Custom fields as key-value pairs */
  customFields?: Record<string>;

  /** Creation timestamp (ISO 8601) */
  createdAt: string;

  /** Last update timestamp (ISO 8601) */
  updatedAt: string;
}

/** Request body for creating a contact */
@example(#{
  email: "alice@example.com",
  firstName: "Alice",
  lastName: "Johnson",
  timeZone: "America/New_York",
  customFields: #{
    plan: "pro",
    source: "import",
  },
})
model CreateContactRequest {
  /** Email address */
  email: string;

  /** First name */
  firstName?: string;

  /** Last name */
  lastName?: string;

  /** IANA time zone identifier */
  timeZone?: string;

  /** Custom fields as key-value pairs */
  customFields?: Record<string>;
}

/** Request body for updating a contact */
@example(#{
  firstName: "Alicia",
  timeZone: "Europe/Berlin",
  customFields: #{
    plan: "enterprise",
  },
})
model UpdateContactRequest {
  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: string | null;

  /** Custom fields as key-value pairs */
  customFields?: Record<string> | null;
}

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

/** Spring Data paginated response */
@example(#{
  content: #[
    #{
      id: 101,
      email: "alice@example.com",
      firstName: "Alice",
      lastName: "Johnson",
      status: ContactStatus.active,
      timeZone: "America/New_York",
      customFields: #{
        plan: "pro",
        source: "import",
      },
      createdAt: "2026-04-01T10:15:30Z",
      updatedAt: "2026-04-03T08:45:00Z",
    },
  ],
  pageable: #{
    pageSize: 25,
    pageNumber: 0,
    offset: 0,
    paged: true,
    unpaged: false,
    sort: #{
      sorted: true,
      unsorted: false,
      empty: false,
    },
  },
  totalElements: 1,
  totalPages: 1,
  size: 25,
  number: 0,
  numberOfElements: 1,
  first: true,
  last: true,
  empty: false,
  sort: #{
    sorted: true,
    unsorted: false,
    empty: false,
  },
})
model ContactPageResponse {
  /** List of contacts */
  content: ContactResponse[];

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

alias ValidationErrors = Record<string[]>;

/** RFC 7807 Problem Details */
@example(#{
  type: "https://onemail.dev/problems/validation-error",
  title: "Validation error",
  status: 422,
  detail: "Request validation failed",
  instance: "/api/contacts",
  errors: #{
    email: #["must be a well-formed email address"],
  },
})
model ProblemDetails {
  /** A URI reference that identifies the problem type */
  type?: string;

  /** A short, human-readable summary of the problem type */
  title?: string;

  /** HTTP status code */
  status?: int32;

  /** A human-readable explanation specific to this occurrence */
  detail?: string;

  /** A URI reference that identifies the specific occurrence */
  instance?: string;

  /** Validation errors grouped by field */
  errors?: ValidationErrors;
}

/** RFC 7807 bad request response */
@error
model BadRequestProblem is Response<400> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 unauthorized response */
@error
model UnauthorizedProblem is Response<401> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 conflict response */
@error
model ConflictProblem is Response<409> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 not found response */
@error
model NotFoundProblem is Response<404> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 validation response */
@error
model UnprocessableEntityProblem is Response<422> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

// ─── Contacts API ───────────────────────────────────────────────────────────

@route("/contacts")
@tag("Contacts")
namespace Contacts {
  /** Create a contact */
  @post
  @opExample(#{
    parameters: #{
      body: #{
        email: "alice@example.com",
        firstName: "Alice",
        lastName: "Johnson",
        timeZone: "America/New_York",
        customFields: #{
          plan: "pro",
          source: "import",
        },
      },
    },
    returnType: #{
      statusCode: 201,
      body: #{
        id: 101,
        email: "alice@example.com",
        firstName: "Alice",
        lastName: "Johnson",
        status: ContactStatus.active,
        timeZone: "America/New_York",
        customFields: #{
          plan: "pro",
          source: "import",
        },
        createdAt: "2026-04-01T10:15:30Z",
        updatedAt: "2026-04-03T08:45:00Z",
      },
    },
  }, #{ title: "Create contact" })
  op create(@body body: CreateContactRequest): {
    @statusCode statusCode: 201;
    @body body: ContactResponse;
  } | UnauthorizedProblem | ConflictProblem | UnprocessableEntityProblem;

  /** List contacts with pagination */
  @get
  @opExample(#{
    parameters: #{
      page: 0,
      size: 25,
      sort: #["email,asc"],
      status: ContactStatus.active,
    },
    returnType: #{
      content: #[
        #{
          id: 101,
          email: "alice@example.com",
          firstName: "Alice",
          lastName: "Johnson",
          status: ContactStatus.active,
          timeZone: "America/New_York",
          customFields: #{
            plan: "pro",
            source: "import",
          },
          createdAt: "2026-04-01T10:15:30Z",
          updatedAt: "2026-04-03T08:45:00Z",
        },
      ],
      pageable: #{
        pageSize: 25,
        pageNumber: 0,
        offset: 0,
        paged: true,
        unpaged: false,
        sort: #{
          sorted: true,
          unsorted: false,
          empty: false,
        },
      },
      totalElements: 1,
      totalPages: 1,
      size: 25,
      number: 0,
      numberOfElements: 1,
      first: true,
      last: true,
      empty: false,
      sort: #{
        sorted: true,
        unsorted: false,
        empty: false,
      },
    },
  }, #{ title: "List contacts" })
  op list(
    /** Page number (0-based) */
    @query page?: int32 = 0,

    /** Page size */
    @query size?: int32 = 25,

    /** Sorting criteria (e.g. name,asc). Can be repeated. */
    @query sort?: string[],

    /** Filter by status */
    @query status?: ContactStatus,
  ): ContactPageResponse | BadRequestProblem | UnauthorizedProblem | UnprocessableEntityProblem;

  /** Get a contact by ID */
  @get
  @route("/{id}")
  @opExample(#{
    parameters: #{
      id: 101,
    },
    returnType: #{
      id: 101,
      email: "alice@example.com",
      firstName: "Alice",
      lastName: "Johnson",
      status: ContactStatus.active,
      timeZone: "America/New_York",
      customFields: #{
        plan: "pro",
        source: "import",
      },
      createdAt: "2026-04-01T10:15:30Z",
      updatedAt: "2026-04-03T08:45:00Z",
    },
  }, #{ title: "Get contact" })
  op get(@path id: int64): ContactResponse | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;

  /** Update a contact */
  @put
  @route("/{id}")
  @opExample(#{
    parameters: #{
      id: 101,
      body: #{
        firstName: "Alicia",
        timeZone: "Europe/Berlin",
        customFields: #{
          plan: "enterprise",
        },
      },
    },
    returnType: #{
      id: 101,
      email: "alice@example.com",
      firstName: "Alicia",
      lastName: "Johnson",
      status: ContactStatus.active,
      timeZone: "Europe/Berlin",
      customFields: #{
        plan: "enterprise",
      },
      createdAt: "2026-04-01T10:15:30Z",
      updatedAt: "2026-04-06T12:00:00Z",
    },
  }, #{ title: "Update contact" })
  op update(@path id: int64, @body body: UpdateContactRequest): ContactResponse | BadRequestProblem | UnauthorizedProblem | NotFoundProblem | ConflictProblem | UnprocessableEntityProblem;

  /** Delete a contact */
  @delete
  @route("/{id}")
  op delete(@path id: int64): NoContentResponse | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;
}

// ─── Segments API ───────────────────────────────────────────────────────────

/** Segment type */
enum SegmentType {
  rule,
  snapshot,
}

/** Segment resource */
@example(#{
  id: 301,
  name: "Recent signups",
  type: SegmentType.rule,
  definition: "createdAt >= now() - P30D",
  createdAt: "2026-04-01T12:00:00Z",
  updatedAt: "2026-04-03T09:00:00Z",
})
model Segment {
  /** Unique identifier */
  id: int64;

  /** Segment name */
  name: string;

  /** Segment type */
  type: SegmentType;

  /** Segment definition (rule or descriptor) */
  definition?: string;

  /** Creation timestamp (ISO 8601) */
  createdAt: string;

  /** Last update timestamp (ISO 8601) */
  updatedAt: string;
}

/** Request body for creating a segment */
@example(#{
  name: "Recent signups",
  type: SegmentType.rule,
  definition: "createdAt >= now() - P30D",
})
model CreateSegmentRequest {
  /** Segment name */
  name: string;

  /** Segment type */
  type: SegmentType;

  /** Segment definition (required for rule segments) */
  definition?: string;
}

/** Request body for updating a segment */
@example(#{
  name: "Warm leads",
  definition: "score >= 50",
})
model UpdateSegmentRequest {
  /** Segment name */
  name?: string | null;

  /** Segment type */
  type?: SegmentType | null;

  /** Segment definition */
  definition?: string | null;
}

/** Spring Data paginated response */
@example(#{
  content: #[
    #{
      id: 301,
      name: "Recent signups",
      type: SegmentType.rule,
      definition: "createdAt >= now() - P30D",
      createdAt: "2026-04-01T12:00:00Z",
      updatedAt: "2026-04-03T09:00:00Z",
    },
  ],
  pageable: #{
    pageSize: 25,
    pageNumber: 0,
    offset: 0,
    paged: true,
    unpaged: false,
    sort: #{
      sorted: true,
      unsorted: false,
      empty: false,
    },
  },
  totalElements: 1,
  totalPages: 1,
  size: 25,
  number: 0,
  numberOfElements: 1,
  first: true,
  last: true,
  empty: false,
  sort: #{
    sorted: true,
    unsorted: false,
    empty: false,
  },
})
model SegmentPage {
  /** List of segments */
  content: Segment[];

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

@route("/segments")
@tag("Segments")
namespace Segments {
  /** Create a segment */
  @post
  @opExample(#{
    parameters: #{
      body: #{
        name: "Recent signups",
        type: SegmentType.rule,
        definition: "createdAt >= now() - P30D",
      },
    },
    returnType: #{
      statusCode: 201,
      body: #{
        id: 301,
        name: "Recent signups",
        type: SegmentType.rule,
        definition: "createdAt >= now() - P30D",
        createdAt: "2026-04-01T12:00:00Z",
        updatedAt: "2026-04-03T09:00:00Z",
      },
    },
  }, #{ title: "Create segment" })
  op create(@body body: CreateSegmentRequest): {
    @statusCode statusCode: 201;
    @body body: Segment;
  } | UnauthorizedProblem | UnprocessableEntityProblem;

  /** List segments with pagination */
  @get
  @opExample(#{
    parameters: #{
      page: 0,
      size: 25,
      sort: #["name,asc"],
    },
    returnType: #{
      content: #[
        #{
          id: 301,
          name: "Recent signups",
          type: SegmentType.rule,
          definition: "createdAt >= now() - P30D",
          createdAt: "2026-04-01T12:00:00Z",
          updatedAt: "2026-04-03T09:00:00Z",
        },
      ],
      pageable: #{
        pageSize: 25,
        pageNumber: 0,
        offset: 0,
        paged: true,
        unpaged: false,
        sort: #{
          sorted: true,
          unsorted: false,
          empty: false,
        },
      },
      totalElements: 1,
      totalPages: 1,
      size: 25,
      number: 0,
      numberOfElements: 1,
      first: true,
      last: true,
      empty: false,
      sort: #{
        sorted: true,
        unsorted: false,
        empty: false,
      },
    },
  }, #{ title: "List segments" })
  op list(
    /** Page number (0-based) */
    @query page?: int32 = 0,

    /** Page size */
    @query size?: int32 = 25,

    /** Sorting criteria (e.g. name,asc). Can be repeated. */
    @query sort?: string[],
  ): SegmentPage | BadRequestProblem | UnauthorizedProblem | UnprocessableEntityProblem;

  /** Get a segment by ID */
  @get
  @route("/{id}")
  @opExample(#{
    parameters: #{
      id: 301,
    },
    returnType: #{
      id: 301,
      name: "Recent signups",
      type: SegmentType.rule,
      definition: "createdAt >= now() - P30D",
      createdAt: "2026-04-01T12:00:00Z",
      updatedAt: "2026-04-03T09:00:00Z",
    },
  }, #{ title: "Get segment" })
  op get(@path id: int64): Segment | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;

  /** Update a segment */
  @put
  @route("/{id}")
  @opExample(#{
    parameters: #{
      id: 301,
      body: #{
        name: "Warm leads",
        definition: "score >= 50",
      },
    },
    returnType: #{
      id: 301,
      name: "Warm leads",
      type: SegmentType.rule,
      definition: "score >= 50",
      createdAt: "2026-04-01T12:00:00Z",
      updatedAt: "2026-04-06T12:00:00Z",
    },
  }, #{ title: "Update segment" })
  op update(@path id: int64, @body body: UpdateSegmentRequest): Segment | BadRequestProblem | UnauthorizedProblem | NotFoundProblem | UnprocessableEntityProblem;

  /** Delete a segment */
  @delete
  @route("/{id}")
  op delete(@path id: int64): NoContentResponse | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;
}

// ─── Broadcasts API ─────────────────────────────────────────────────────────

/** Broadcast status */
enum BroadcastStatus {
  draft,
  scheduled,
  sent,
}

/** Broadcast resource */
@example(#{
  id: 501,
  name: "April product update",
  segmentId: 301,
  status: BroadcastStatus.draft,
  createdAt: "2026-04-02T08:00:00Z",
  updatedAt: "2026-04-04T16:30:00Z",
})
model Broadcast {
  /** Unique identifier */
  id: int64;

  /** Broadcast name */
  name: string;

  /** Segment ID */
  segmentId: int64;

  /** Broadcast status */
  status: BroadcastStatus;

  /** Creation timestamp (ISO 8601) */
  createdAt: string;

  /** Last update timestamp (ISO 8601) */
  updatedAt: string;
}

/** Request body for creating a broadcast */
@example(#{
  name: "April product update",
  segmentId: 301,
  status: BroadcastStatus.draft,
})
model CreateBroadcastRequest {
  /** Broadcast name */
  name: string;

  /** Segment ID */
  segmentId: int64;

  /** Broadcast status */
  status: BroadcastStatus;
}

/** Request body for updating a broadcast */
@example(#{
  name: "April product update v2",
  segmentId: 301,
  status: BroadcastStatus.scheduled,
})
model UpdateBroadcastRequest {
  /** Broadcast name */
  name?: string | null;

  /** Segment ID */
  segmentId?: int64 | null;

  /** Broadcast status */
  status?: BroadcastStatus | null;
}

/** Spring Data paginated response */
@example(#{
  content: #[
    #{
      id: 501,
      name: "April product update",
      segmentId: 301,
      status: BroadcastStatus.draft,
      createdAt: "2026-04-02T08:00:00Z",
      updatedAt: "2026-04-04T16:30:00Z",
    },
  ],
  pageable: #{
    pageSize: 25,
    pageNumber: 0,
    offset: 0,
    paged: true,
    unpaged: false,
    sort: #{
      sorted: true,
      unsorted: false,
      empty: false,
    },
  },
  totalElements: 1,
  totalPages: 1,
  size: 25,
  number: 0,
  numberOfElements: 1,
  first: true,
  last: true,
  empty: false,
  sort: #{
    sorted: true,
    unsorted: false,
    empty: false,
  },
})
model BroadcastPage {
  /** List of broadcasts */
  content: Broadcast[];

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

@route("/broadcasts")
@tag("Broadcasts")
namespace Broadcasts {
  /** Create a broadcast */
  @post
  @opExample(#{
    parameters: #{
      body: #{
        name: "April product update",
        segmentId: 301,
        status: BroadcastStatus.draft,
      },
    },
    returnType: #{
      statusCode: 201,
      body: #{
        id: 501,
        name: "April product update",
        segmentId: 301,
        status: BroadcastStatus.draft,
        createdAt: "2026-04-02T08:00:00Z",
        updatedAt: "2026-04-04T16:30:00Z",
      },
    },
  }, #{ title: "Create broadcast" })
  op create(@body body: CreateBroadcastRequest): {
    @statusCode statusCode: 201;
    @body body: Broadcast;
  } | UnauthorizedProblem | UnprocessableEntityProblem;

  /** List broadcasts with pagination */
  @get
  @opExample(#{
    parameters: #{
      page: 0,
      size: 25,
      sort: #["createdAt,desc"],
    },
    returnType: #{
      content: #[
        #{
          id: 501,
          name: "April product update",
          segmentId: 301,
          status: BroadcastStatus.draft,
          createdAt: "2026-04-02T08:00:00Z",
          updatedAt: "2026-04-04T16:30:00Z",
        },
      ],
      pageable: #{
        pageSize: 25,
        pageNumber: 0,
        offset: 0,
        paged: true,
        unpaged: false,
        sort: #{
          sorted: true,
          unsorted: false,
          empty: false,
        },
      },
      totalElements: 1,
      totalPages: 1,
      size: 25,
      number: 0,
      numberOfElements: 1,
      first: true,
      last: true,
      empty: false,
      sort: #{
        sorted: true,
        unsorted: false,
        empty: false,
      },
    },
  }, #{ title: "List broadcasts" })
  op list(
    /** Page number (0-based) */
    @query page?: int32 = 0,

    /** Page size */
    @query size?: int32 = 25,

    /** Sorting criteria (e.g. name,asc). Can be repeated. */
    @query sort?: string[],
  ): BroadcastPage | BadRequestProblem | UnauthorizedProblem | UnprocessableEntityProblem;

  /** Get a broadcast by ID */
  @get
  @route("/{id}")
  @opExample(#{
    parameters: #{
      id: 501,
    },
    returnType: #{
      id: 501,
      name: "April product update",
      segmentId: 301,
      status: BroadcastStatus.draft,
      createdAt: "2026-04-02T08:00:00Z",
      updatedAt: "2026-04-04T16:30:00Z",
    },
  }, #{ title: "Get broadcast" })
  op get(@path id: int64): Broadcast | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;

  /** Update a broadcast */
  @put
  @route("/{id}")
  @opExample(#{
    parameters: #{
      id: 501,
      body: #{
        name: "April product update v2",
        segmentId: 301,
        status: BroadcastStatus.scheduled,
      },
    },
    returnType: #{
      id: 501,
      name: "April product update v2",
      segmentId: 301,
      status: BroadcastStatus.scheduled,
      createdAt: "2026-04-02T08:00:00Z",
      updatedAt: "2026-04-06T12:00:00Z",
    },
  }, #{ title: "Update broadcast" })
  op update(@path id: int64, @body body: UpdateBroadcastRequest): Broadcast | BadRequestProblem | UnauthorizedProblem | NotFoundProblem | UnprocessableEntityProblem;

  /** Delete a broadcast */
  @delete
  @route("/{id}")
  op delete(@path id: int64): NoContentResponse | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;
}

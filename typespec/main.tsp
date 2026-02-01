import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

@service(#{})
@route("/api")
namespace OneMail;

// ─── Auth ───────────────────────────────────────────────────────────────────

@useAuth(BearerAuth)
namespace Authenticated {
}

// ─── Models ─────────────────────────────────────────────────────────────────

/** Contact status */
enum ContactStatus {
  active,
  unsubscribed,
}

/** Contact resource */
model Contact {
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
model ContactPage {
  /** List of contacts */
  content: Contact[];

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

/** RFC 7807 Problem Details */
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
  errors?: Record<string, string[]>;
}

/** RFC 7807 Problem Details response */
@error
model ProblemDetailsResponse {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

// ─── Contacts API ───────────────────────────────────────────────────────────

@route("/contacts")
@tag("Contacts")
namespace Contacts {
  /** Create a contact */
  @post
  op create(@body body: CreateContactRequest): {
    @statusCode statusCode: 201;
    @body body: Contact;
  } | {
    @statusCode statusCode: 422;
    @body body: ProblemDetails;
  } | ProblemDetailsResponse;

  /** List contacts with pagination */
  @get
  op list(
    /** Page number (0-based) */
    @query page?: int32 = 0,

    /** Page size */
    @query size?: int32 = 25,

    /** Sorting criteria (e.g. name,asc). Can be repeated. */
    @query sort?: string[],

    /** Filter by status */
    @query status?: ContactStatus,
  ): ContactPage | ProblemDetailsResponse | {
    @statusCode statusCode: 422;
    @body body: ProblemDetails;
  };

  /** Get a contact by ID */
  @get
  @route("/{id}")
  op get(@path id: int64): Contact | ProblemDetailsResponse;

  /** Update a contact */
  @put
  @route("/{id}")
  op update(@path id: int64, @body body: UpdateContactRequest): Contact | ProblemDetailsResponse | {
    @statusCode statusCode: 422;
    @body body: ProblemDetails;
  };

  /** Delete a contact */
  @delete
  @route("/{id}")
  op delete(@path id: int64): {
    @statusCode statusCode: 204;
  } | ProblemDetailsResponse;
}

// ─── Segments API ───────────────────────────────────────────────────────────

/** Segment type */
enum SegmentType {
  rule,
  snapshot,
}

/** Segment resource */
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
model CreateSegmentRequest {
  /** Segment name */
  name: string;

  /** Segment type */
  type: SegmentType;

  /** Segment definition (required for rule segments) */
  definition?: string;
}

/** Request body for updating a segment */
model UpdateSegmentRequest {
  /** Segment name */
  name?: string | null;

  /** Segment type */
  type?: SegmentType | null;

  /** Segment definition */
  definition?: string | null;
}

/** Spring Data paginated response */
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
  op create(@body body: CreateSegmentRequest): {
    @statusCode statusCode: 201;
    @body body: Segment;
  } | {
    @statusCode statusCode: 422;
    @body body: ProblemDetails;
  } | ProblemDetailsResponse;

  /** List segments with pagination */
  @get
  op list(
    /** Page number (0-based) */
    @query page?: int32 = 0,

    /** Page size */
    @query size?: int32 = 25,

    /** Sorting criteria (e.g. name,asc). Can be repeated. */
    @query sort?: string[],
  ): SegmentPage | ProblemDetailsResponse | {
    @statusCode statusCode: 422;
    @body body: ProblemDetails;
  };

  /** Get a segment by ID */
  @get
  @route("/{id}")
  op get(@path id: int64): Segment | ProblemDetailsResponse;

  /** Update a segment */
  @put
  @route("/{id}")
  op update(@path id: int64, @body body: UpdateSegmentRequest): Segment | ProblemDetailsResponse | {
    @statusCode statusCode: 422;
    @body body: ProblemDetails;
  };

  /** Delete a segment */
  @delete
  @route("/{id}")
  op delete(@path id: int64): {
    @statusCode statusCode: 204;
  } | ProblemDetailsResponse;
}

// ─── Broadcasts API ─────────────────────────────────────────────────────────

/** Broadcast status */
enum BroadcastStatus {
  draft,
  scheduled,
  sent,
}

/** Broadcast resource */
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
model CreateBroadcastRequest {
  /** Broadcast name */
  name: string;

  /** Segment ID */
  segmentId: int64;

  /** Broadcast status */
  status: BroadcastStatus;
}

/** Request body for updating a broadcast */
model UpdateBroadcastRequest {
  /** Broadcast name */
  name?: string | null;

  /** Segment ID */
  segmentId?: int64 | null;

  /** Broadcast status */
  status?: BroadcastStatus | null;
}

/** Spring Data paginated response */
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
  op create(@body body: CreateBroadcastRequest): {
    @statusCode statusCode: 201;
    @body body: Broadcast;
  } | {
    @statusCode statusCode: 422;
    @body body: ProblemDetails;
  } | ProblemDetailsResponse;

  /** List broadcasts with pagination */
  @get
  op list(
    /** Page number (0-based) */
    @query page?: int32 = 0,

    /** Page size */
    @query size?: int32 = 25,

    /** Sorting criteria (e.g. name,asc). Can be repeated. */
    @query sort?: string[],
  ): BroadcastPage | ProblemDetailsResponse | {
    @statusCode statusCode: 422;
    @body body: ProblemDetails;
  };

  /** Get a broadcast by ID */
  @get
  @route("/{id}")
  op get(@path id: int64): Broadcast | ProblemDetailsResponse;

  /** Update a broadcast */
  @put
  @route("/{id}")
  op update(@path id: int64, @body body: UpdateBroadcastRequest): Broadcast | ProblemDetailsResponse | {
    @statusCode statusCode: 422;
    @body body: ProblemDetails;
  };

  /** Delete a broadcast */
  @delete
  @route("/{id}")
  op delete(@path id: int64): {
    @statusCode statusCode: 204;
  } | ProblemDetailsResponse;
}

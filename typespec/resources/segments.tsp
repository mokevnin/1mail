using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Segment type */
enum SegmentType {
  rule,
  snapshot,
}

/** Segment resource */
@resource("segments")
@segment("segments")
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
  @key
  id: int64;

  /** Segment name */
  name: string;

  /** Segment type */
  type: SegmentType;

  /** Segment definition (rule or descriptor) */
  definition?: string;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating a segment */
@example(#{
  name: "Recent signups",
  type: SegmentType.rule,
  definition: "createdAt >= now() - P30D",
})
model CreateSegmentInput {
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
model UpdateSegmentInput {
  /** Segment name */
  name?: string | null;

  /** Segment type */
  type?: SegmentType | null;

  /** Segment definition */
  definition?: string | null;
}

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
model SegmentPage is Page<Segment> {}

@route("/segments")
@tag("Segments")
namespace Segments {
  /** Create a segment */
  @createsResource(Segment)
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
  op create(@body body: CreateSegmentInput): {
    @statusCode statusCode: 201;
    @body body: Segment;
  } | UnauthorizedProblem | UnprocessableEntityProblem;

  /** List segments with pagination */
  @listsResource(Segment)
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
  op list(...PageQuery): SegmentPage | BadRequestProblem | UnauthorizedProblem | UnprocessableEntityProblem;

  /** Get a segment by ID */
  @readsResource(Segment)
  @get
  @autoRoute
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
  @updatesResource(Segment)
  @put
  @autoRoute
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
  op update(@path id: int64, @body body: UpdateSegmentInput): Segment | BadRequestProblem | UnauthorizedProblem | NotFoundProblem | UnprocessableEntityProblem;

  /** Delete a segment */
  @deletesResource(Segment)
  @delete
  @autoRoute
  op delete(@path id: int64): NoContentResponse | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;
}

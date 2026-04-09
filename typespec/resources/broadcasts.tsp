using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Broadcast status */
enum BroadcastStatus {
  draft,
  scheduled,
  sent,
}

/** Broadcast resource */
@resource("broadcasts")
@segment("broadcasts")
@example(#{
  id: 501,
  name: "April product update",
  segmentId: 301,
  status: BroadcastStatus.draft,
  createdAt: "2026-04-02T08:00:00Z",
  updatedAt: "2026-04-04T16:30:00Z",
})
model BroadcastResource {
  /** Unique identifier */
  @key
  id: int64;

  /** Broadcast name */
  name: string;

  /** Segment ID */
  segmentId: int64;

  /** Broadcast status */
  status: BroadcastStatus;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating a broadcast */
@example(#{
  name: "April product update",
  segmentId: 301,
  status: BroadcastStatus.draft,
})
model CreateBroadcastInput {
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
model UpdateBroadcastInput {
  /** Broadcast name */
  name?: string | null;

  /** Segment ID */
  segmentId?: int64 | null;

  /** Broadcast status */
  status?: BroadcastStatus | null;
}

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
model BroadcastPage is Page<BroadcastResource> {}

@route("/broadcasts")
@tag("Broadcasts")
namespace Broadcasts {
  /** Create a broadcast */
  @createsResource(BroadcastResource)
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
  op create(@body body: CreateBroadcastInput): {
    @statusCode statusCode: 201;
    @body body: BroadcastResource;
  } | UnauthorizedProblem | UnprocessableEntityProblem;

  /** List broadcasts with pagination */
  @listsResource(BroadcastResource)
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
  op list(...PageQuery): BroadcastPage | BadRequestProblem | UnauthorizedProblem | UnprocessableEntityProblem;

  /** Get a broadcast by ID */
  @readsResource(BroadcastResource)
  @get
  @autoRoute
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
  op get(@path id: int64): BroadcastResource | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;

  /** Update a broadcast */
  @updatesResource(BroadcastResource)
  @put
  @autoRoute
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
  op update(@path id: int64, @body body: UpdateBroadcastInput): BroadcastResource | BadRequestProblem | UnauthorizedProblem | NotFoundProblem | UnprocessableEntityProblem;

  /** Delete a broadcast */
  @deletesResource(BroadcastResource)
  @delete
  @autoRoute
  op delete(@path id: int64): NoContentResponse | BadRequestProblem | UnauthorizedProblem | NotFoundProblem;
}

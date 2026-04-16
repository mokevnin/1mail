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
model BroadcastResource extends CreateBroadcastInput {
  /** Unique identifier */
  @key
  id: int64;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating a broadcast */
model CreateBroadcastInput {
  /** Broadcast name */
  name: string;

  /** Segment ID */
  segmentId: int64;

  /** Broadcast status */
  status: BroadcastStatus;
}

/** Request body for updating a broadcast */
model UpdateBroadcastInput {
  /** Broadcast name */
  name?: string;

  /** Segment ID */
  segmentId?: int64;

  /** Broadcast status */
  status?: BroadcastStatus;
}

model BroadcastPage is Page<BroadcastResource>;

@route("/broadcasts")
@tag("Broadcasts")
namespace Broadcasts {
  /** Create a broadcast */
  @createsResource(BroadcastResource)
  @post
  op create(@body body: CreateBroadcastInput):
    | {
        @statusCode statusCode: 201;
        @body body: BroadcastResource;
      }
    | UnauthorizedProblem
    | UnprocessableEntityProblem;

  /** List broadcasts with pagination */
  @listsResource(BroadcastResource)
  @get
  op list(...PageQuery):
    | BroadcastPage
    | BadRequestProblem
    | UnauthorizedProblem
    | UnprocessableEntityProblem;

  /** Get a broadcast by ID */
  @readsResource(BroadcastResource)
  @get
  @autoRoute
  op get(@path id: int64):
    | BroadcastResource
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem;

  /** Update a broadcast */
  @updatesResource(BroadcastResource)
  @put
  @autoRoute
  op update(@path id: int64, @body body: UpdateBroadcastInput):
    | BroadcastResource
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Delete a broadcast */
  @deletesResource(BroadcastResource)
  @delete
  @autoRoute
  op delete(@path id: int64):
    | NoContentResponse
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem;
}

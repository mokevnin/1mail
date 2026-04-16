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
model SegmentResource extends CreateSegmentInput {
  /** Unique identifier */
  @key
  id: int64;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating a segment */
model CreateSegmentInput {
  /** Segment name */
  name: string;

  /** Segment type */
  type: SegmentType;

  /** Segment definition (required for rule segments) */
  definition?: string;
}

/** Request body for updating a segment */
model UpdateSegmentInput {
  /** Segment name */
  name?: string;

  /** Segment type */
  type?: SegmentType;

  /** Segment definition */
  definition?: string;
}

model SegmentPage is Page<SegmentResource>;

@route("/segments")
@tag("Segments")
namespace Segments {
  /** Create a segment */
  @createsResource(SegmentResource)
  @post
  op create(@body body: CreateSegmentInput):
    | {
        @statusCode statusCode: 201;
        @body body: SegmentResource;
      }
    | UnauthorizedProblem
    | UnprocessableEntityProblem;

  /** List segments with pagination */
  @listsResource(SegmentResource)
  @get
  op list(...PageQuery):
    | SegmentPage
    | BadRequestProblem
    | UnauthorizedProblem
    | UnprocessableEntityProblem;

  /** Get a segment by ID */
  @readsResource(SegmentResource)
  @get
  @autoRoute
  op get(@path id: int64):
    | SegmentResource
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem;

  /** Update a segment */
  @updatesResource(SegmentResource)
  @put
  @autoRoute
  op update(@path id: int64, @body body: UpdateSegmentInput):
    | SegmentResource
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Delete a segment */
  @deletesResource(SegmentResource)
  @delete
  @autoRoute
  op delete(@path id: int64):
    | NoContentResponse
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem;
}

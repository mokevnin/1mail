using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Segment type for site UI */
enum SiteSegmentType {
  rule,
  snapshot,
}

/** Segment resource used by the site UI */
@resource("segments")
@segment("segments")
model SiteSegmentResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Segment name */
  name: string;

  /** Segment type */
  type: SiteSegmentType;

  /** Segment definition (used by rule segments) */
  definition?: string | null;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for creating a segment */
model SiteCreateSegmentInput {
  /** Segment name */
  name: string;

  /** Segment type */
  type: SiteSegmentType;

  /** Segment definition (used by rule segments) */
  definition?: string | null;
}

/** Site request body for updating a segment */
model SiteUpdateSegmentInput {
  /** Segment name */
  name?: string;

  /** Segment type */
  type?: SiteSegmentType;

  /** Segment definition (used by rule segments) */
  definition?: string | null;
}

@route("/w/{workspaceSlug}/segments")
@tag("Site Segments")
namespace SiteSegments {
  /** Create a segment from the site UI */
  @createsResource(SiteSegmentResource)
  @post
  op create(@path workspaceSlug: string, @body body: SiteCreateSegmentInput):
    | {
        @statusCode statusCode: 201;
        @body body: SiteSegmentResource;
      }
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** List segments for the site UI */
  @listsResource(SiteSegmentResource)
  @get
  op list(@path workspaceSlug: string, ...PageQuery):
    | Page<SiteSegmentResource>
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Get a segment by ID for the site UI */
  @readsResource(SiteSegmentResource)
  @get
  @route("/{id}")
  op get(@path workspaceSlug: string, @path id: EntityId):
    | SiteSegmentResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Update a segment from the site UI */
  @updatesResource(SiteSegmentResource)
  @put
  @route("/{id}")
  op update(
    @path workspaceSlug: string,
    @path id: EntityId,
    @body body: SiteUpdateSegmentInput,
  ):
    | SiteSegmentResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Delete a segment from the site UI */
  @deletesResource(SiteSegmentResource)
  @delete
  @route("/{id}")
  op delete(@path workspaceSlug: string, @path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;
}

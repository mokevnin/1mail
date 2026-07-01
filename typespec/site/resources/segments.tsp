using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

namespace OneMail;

/** Segment type for site UI */
enum SiteSegmentType {
  rule,
  snapshot,
}

/** Segment resource used by the site UI */
@resource("segments")
@segment("segments")
@parentResource(SiteWorkspaceResource)
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

/** Request body for previewing a rule definition's audience */
model SitePreviewSegmentInput {
  /** Rule definition (react-querybuilder JSON); empty means all contacts */
  definition?: string | null;
}

/** Result of a segment preview: how many deliverable contacts match */
model SitePreviewSegmentResult {
  /** Number of active contacts matching the rule */
  count: int32;
}

@tag("Site Segments")
interface SiteSegments
  extends WorkspaceCrud<
      SiteSegmentResource,
      SiteCreateSegmentInput,
      SiteUpdateSegmentInput
    > {
  /** Preview how many active contacts match a rule definition */
  @collectionAction(SiteSegmentResource, "preview")
  @autoRoute
  preview(
    ...ResourceCollectionParameters<SiteSegmentResource>,
    @body body: SitePreviewSegmentInput,
  ):
    | SitePreviewSegmentResult
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;
}

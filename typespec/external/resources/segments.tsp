using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

namespace OneMail;

/** Segment resource */
@resource("segments")
@segment("segments")
model SegmentResource extends CreateSegmentInput {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating a segment */
model CreateSegmentInput {
  /** Segment name */
  name: string;

  /** Segment definition (rule query; empty matches all contacts) */
  definition?: string;
}

/** Request body for updating a segment */
model UpdateSegmentInput {
  /** Segment name */
  name?: string;

  /** Segment definition */
  definition?: string;
}

/** Request body for previewing a rule definition's audience */
model PreviewSegmentInput {
  /** Rule definition (react-querybuilder JSON); empty means all contacts */
  definition?: string | null;
}

/** Result of a segment preview */
model PreviewSegmentResult {
  /** Number of contacts matching the rule */
  count: int32;
}

@tag("Segments")
interface Segments
  extends ExternalCrud<
      SegmentResource,
      CreateSegmentInput,
      UpdateSegmentInput
    > {
  /** Count the contacts matching an unsaved rule definition */
  @collectionAction(SegmentResource, "preview")
  @autoRoute
  preview(
    ...ResourceCollectionParameters<SegmentResource>,
    @body body: PreviewSegmentInput,
  ):
    | PreviewSegmentResult
    | BadRequestProblem
    | UnauthorizedProblem
    | UnprocessableEntityProblem;
}

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

@tag("Segments")
interface Segments
  extends ExternalCrud<
      SegmentResource,
      CreateSegmentInput,
      UpdateSegmentInput
    > {}

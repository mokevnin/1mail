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
  id: EntityId;

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
  segmentId: EntityId;

  /** Broadcast status */
  status: BroadcastStatus;
}

/** Request body for updating a broadcast */
model UpdateBroadcastInput {
  /** Broadcast name */
  name?: string;

  /** Segment ID */
  segmentId?: EntityId;

  /** Broadcast status */
  status?: BroadcastStatus;
}

@tag("Broadcasts")
interface Broadcasts
  extends ExternalCrud<
      BroadcastResource,
      CreateBroadcastInput,
      UpdateBroadcastInput
    > {}

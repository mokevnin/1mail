using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Event payload */
model EventInput {
  /** Canonical subject identifier */
  subjectId: string;

  /** Event action */
  action: string;

  /** Email address associated with the event */
  email?: EmailAddress | null;

  /** Phone number associated with the event */
  phone?: string | null;

  /** Event properties */
  properties?: Record<unknown> | null;

  /** Event occurrence timestamp */
  occurredAt?: Timestamp | null;
}

/** Request body for recording events */
model RecordEventsInput {
  /** Events to record */
  events: EventInput[];
}

/** Request body for recording events one by one with per-item results */
model RecordEventsBatchInput {
  /** Events to record, at most 1000 */
  @minItems(1)
  @maxItems(1000)
  events: EventInput[];
}

/** Outcome of one item of an events batch */
enum EventBatchStatus {
  accepted,
  failed,
}

/** Per-item result of an events batch, in request order */
model EventBatchItemResult {
  /** Zero-based position of the item in the request */
  index: int32;

  /** accepted when published; failed when the item was rejected */
  status: EventBatchStatus;

  /** Why the item failed; absent when accepted */
  error?: string;
}

/** Per-item results of an events batch */
model RecordEventsBatchResult {
  /** One result per request item, in request order */
  results: EventBatchItemResult[];
}

/** Distinct event action */
model EventActionResource {
  /** Event action */
  action: string;
}

@route("/events")
@tag("Events")
namespace Events {
  /** Record events in batch */
  @post
  op create(@body body: RecordEventsInput):
    | NoContentResponse
    | UnauthorizedProblem
    | UnprocessableEntityProblem;
}

// Aligned with the site API's `/events/actions` naming.
@route("/events/actions")
@tag("Events")
namespace EventActions {
  /** List unique event actions */
  @get
  op list(...PageQuery):
    | Page<EventActionResource>
    | BadRequestProblem
    | UnauthorizedProblem
    | UnprocessableEntityProblem;
}

@route("/events/batch")
@tag("Events")
namespace EventsBatch {
  /** Record up to 1000 events; a failing item does not affect the others */
  @post
  op submit(@body body: RecordEventsBatchInput):
    | RecordEventsBatchResult
    | UnauthorizedProblem
    | UnprocessableEntityProblem;
}

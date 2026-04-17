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

  /** Prospect flag */
  prospect?: boolean | null;

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
  op create(
    @body body: RecordEventsInput,
  ): NoContentResponse | UnauthorizedProblem | UnprocessableEntityProblem;
}

@route("/event-actions")
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

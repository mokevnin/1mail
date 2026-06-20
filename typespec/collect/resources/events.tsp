using TypeSpec.Http;

namespace OneMail;

model CollectEventInput {
  visitorId: string;
  action: string;
  properties?: Record<unknown> | null;
  occurredAt?: Timestamp | null;
}

model CollectEventsInput {
  events: CollectEventInput[];
}

@route("/events")
@tag("Collect")
namespace CollectEvents {
  @post
  op create(@body body: CollectEventsInput):
    | NoContentResponse
    | UnauthorizedProblem
    | UnprocessableEntityProblem;
}

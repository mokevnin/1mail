using TypeSpec.Http;

namespace OneMail;

model CollectIdentifyInput {
  visitorId: string;
  email?: EmailAddress | null;
  phone?: string | null;
  subjectId?: string | null;
  traits?: Record<unknown> | null;
}

model CollectEventInput {
  visitorId: string;
  action: string;
  properties?: Record<unknown> | null;
  occurredAt?: Timestamp | null;
}

model CollectEventsInput {
  events: CollectEventInput[];
}

model CollectOkResponse {
  ok: true;
}

@route("/collect/identify")
@tag("Collect")
namespace CollectIdentify {
  @post
  op create(
    @header("x-collect-key") collectKey: string,
    @body body: CollectIdentifyInput,
  ): CollectOkResponse | UnauthorizedProblem | UnprocessableEntityProblem;
}

@route("/collect/events")
@tag("Collect")
namespace CollectEvents {
  @post
  op create(
    @header("x-collect-key") collectKey: string,
    @body body: CollectEventsInput,
  ): NoContentResponse | UnauthorizedProblem | UnprocessableEntityProblem;
}

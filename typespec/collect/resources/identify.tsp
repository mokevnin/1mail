using TypeSpec.Http;

namespace OneMail;

model CollectIdentifyInput {
  visitorId: string;
  email?: EmailAddress | null;
  phone?: string | null;
  subjectId?: string | null;
  traits?: Record<unknown> | null;
}

model CollectOkResponse {
  ok: true;
}

@route("/identify")
@tag("Collect")
namespace CollectIdentify {
  @post
  op create(
    @body body: CollectIdentifyInput,
  ): CollectOkResponse | UnauthorizedProblem | UnprocessableEntityProblem;
}

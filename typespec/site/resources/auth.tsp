using TypeSpec.Http;

namespace OneMail;

model SiteRegisterInput {
  name: string;
  email: EmailAddress;
  password: string;
}

model SiteRegisterResult {
  id: EntityId;
  name: string;
  email: EmailAddress;
  createdAt: Timestamp;
}

@route("/auth")
@tag("Site Auth")
namespace SiteAuth {
  @post
  @route("/register")
  op register(@body body: SiteRegisterInput): {
    @statusCode statusCode: 201;
    @body body: SiteRegisterResult;
  } | ConflictProblem | UnprocessableEntityProblem;
}

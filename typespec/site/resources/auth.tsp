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

model SiteDirectLoginInput {
  user: string;
  passwd: string;
}

model SiteDirectLoginResult {
  name: string;
  id: string;
  picture?: string;
  aud?: string;
  ip?: string;
  email?: string;
  attrs?: Record<unknown>;
  role?: string;
}

model SiteDirectLoginError {
  error: string;
}

@error
model SiteDirectLoginForbidden is Response<403> {
  @header contentType: "application/json";
  @body body: SiteDirectLoginError;
}

@route("/auth")
@tag("Site Auth")
namespace SiteAuth {
  @useAuth(NoAuth)
  @post
  @route("/register")
  op register(@body body: SiteRegisterInput):
    | {
        @statusCode statusCode: 201;
        @body body: SiteRegisterResult;
      }
    | ConflictProblem
    | UnprocessableEntityProblem;

  @useAuth(NoAuth)
  @post
  @route("/direct/login")
  op directLogin(@body body: SiteDirectLoginInput):
    | {
        @statusCode statusCode: 200;
        @body body: SiteDirectLoginResult;
      }
    | SiteDirectLoginForbidden
    | BadRequestProblem;
}

using TypeSpec.Http;

namespace OneMail;

enum ApiTokenScope {
  contactsRead: "contacts:read",
  contactsWrite: "contacts:write",
  segmentsRead: "segments:read",
  segmentsWrite: "segments:write",
  broadcastsRead: "broadcasts:read",
  broadcastsWrite: "broadcasts:write",
  tokensManage: "tokens:manage",
}

model ApiTokenInfo {
  id: EntityId;
  name: string;
  scopes: ApiTokenScope[];
  expiresAt?: Timestamp | null;
  revokedAt?: Timestamp | null;
  lastUsedAt?: Timestamp | null;
  createdAt: Timestamp;
  updatedAt: Timestamp;
}

model ApiTokenListResponse {
  items: ApiTokenInfo[];
}

model CreateApiTokenInput {
  name: string;
  scopes: ApiTokenScope[];
  expiresAt?: Timestamp | null;
}

model CreateApiTokenResponse {
  token: string;
  tokenInfo: ApiTokenInfo;
}

@route("/auth")
@tag("Auth")
namespace Auth {
  @post
  @route("/tokens")
  op tokensCreate(@body body: CreateApiTokenInput):
    | {
        @statusCode statusCode: 201;
        @body body: CreateApiTokenResponse;
      }
    | UnauthorizedProblem
    | ForbiddenProblem
    | BadRequestProblem;

  @get
  @route("/tokens")
  op tokensList(
  ): ApiTokenListResponse | UnauthorizedProblem | ForbiddenProblem;

  @delete
  @route("/tokens/{id}")
  op tokensDelete(@path id: EntityId):
    | NoContentResponse
    | UnauthorizedProblem
    | ForbiddenProblem
    | NotFoundProblem
    | BadRequestProblem;

  @get
  @route("/me")
  op meGet(): ApiTokenInfo | UnauthorizedProblem | NotFoundProblem;

  @post
  @route("/tokens/bootstrap")
  @useAuth([])
  op tokensBootstrap(
    @header("X-Bootstrap-Token") bootstrapToken: string,
    @body body: CreateApiTokenInput,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: CreateApiTokenResponse;
      }
    | UnauthorizedProblem
    | ForbiddenProblem
    | BadRequestProblem;
}

using TypeSpec.Http;

namespace OneMail;

enum ApiTokenScope {
  contactsRead: "contacts:read",
  contactsWrite: "contacts:write",
  contactsErase: "contacts:erase",
  segmentsRead: "segments:read",
  segmentsWrite: "segments:write",
  eventsRead: "events:read",
  eventsWrite: "events:write",
  broadcastsRead: "broadcasts:read",
  broadcastsWrite: "broadcasts:write",
  automationsRead: "automations:read",
  automationsWrite: "automations:write",
  templatesRead: "templates:read",
  templatesWrite: "templates:write",
  webhooksRead: "webhooks:read",
  webhooksWrite: "webhooks:write",
  customFieldsRead: "custom_fields:read",
  sendingDomainsRead: "sending_domains:read",
  emailsSend: "emails:send",
  broadcastsSend: "broadcasts:send",
  automationsActivate: "automations:activate",
  mcpSend: "mcp:send",
  tokensRead: "tokens:read",
  tokensWrite: "tokens:write",
  auditRead: "audit:read",
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
  op tokensList():
    | ApiTokenListResponse
    | UnauthorizedProblem
    | ForbiddenProblem;

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

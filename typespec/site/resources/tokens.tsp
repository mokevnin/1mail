using TypeSpec.Http;

namespace OneMail;

/** A workspace API token (secret shown only once, at creation) */
model SiteApiTokenResource {
  /** Unique identifier */
  id: EntityId;

  /** Human-readable label */
  name: string;

  /** Public prefix (the secret is never returned after creation) */
  prefix: string;

  /** Granted scopes */
  scopes: string[];

  /** Last time the token authenticated, if ever */
  lastUsedAt?: Timestamp | null;

  /** Expiry, if set */
  expiresAt?: Timestamp | null;

  /** Creation timestamp */
  createdAt: Timestamp;
}

/** Create a workspace API token */
model SiteCreateTokenInput {
  name: string;
  scopes: string[];
  expiresAt?: Timestamp | null;
}

/** Token creation result — `token` is the full secret, shown only once */
model SiteCreateTokenResponse {
  token: string;
  resource: SiteApiTokenResource;
}

@route("/w/{workspaceSlug}/tokens")
@tag("Site Tokens")
namespace SiteTokens {
  /** List active (non-revoked) API tokens for the workspace */
  @get
  op list(@path workspaceSlug: string):
    | {
        @statusCode statusCode: 200;
        @body body: SiteApiTokenResource[];
      }
    | NotFoundProblem;

  /** Create an API token; the full secret is returned once */
  @post
  op create(@path workspaceSlug: string, @body body: SiteCreateTokenInput):
    | {
        @statusCode statusCode: 201;
        @body body: SiteCreateTokenResponse;
      }
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Revoke an API token */
  @delete
  @route("/{id}")
  op delete(@path workspaceSlug: string, @path id: EntityId):
    | NoContentResponse
    | NotFoundProblem
    | BadRequestProblem;
}

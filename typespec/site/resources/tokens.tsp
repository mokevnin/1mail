using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

namespace OneMail;

/** A workspace API token (secret shown only once, at creation) */
@resource("tokens")
@segment("tokens")
@parentResource(SiteWorkspaceResource)
model SiteApiTokenResource {
  /** Unique identifier */
  @key
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

// Bespoke (not WorkspaceCrud): no GET/PUT for a single token, and create returns
// the one-time secret rather than the resource. Nested via @parentResource + @autoRoute.
@tag("Site Tokens")
namespace SiteTokens {
  /** List active (non-revoked) API tokens for the workspace */
  @listsResource(SiteApiTokenResource)
  @get
  @autoRoute
  op list(...ResourceCollectionParameters<SiteApiTokenResource>):
    | {
        @statusCode statusCode: 200;
        @body body: SiteApiTokenResource[];
      }
    | NotFoundProblem;

  /** Create an API token; the full secret is returned once */
  @createsResource(SiteApiTokenResource)
  @post
  @autoRoute
  op create(
    ...ResourceCollectionParameters<SiteApiTokenResource>,
    @body body: SiteCreateTokenInput,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: SiteCreateTokenResponse;
      }
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Revoke an API token */
  @deletesResource(SiteApiTokenResource)
  @delete
  @autoRoute
  op delete(...ResourceParameters<SiteApiTokenResource>):
    | NoContentResponse
    | NotFoundProblem
    | BadRequestProblem;
}

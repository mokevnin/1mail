using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Why an address is on the suppression (do-not-send) list */
enum SiteSuppressionReason {
  unsubscribed,
  bounce,
  complaint,
  manual,
}

/** A suppressed address: the send path skips it regardless of contact status */
@resource("suppressions")
@segment("suppressions")
model SiteSuppressionResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Normalized (lower-cased) email address */
  email: string;

  /** Why the address is suppressed */
  reason: SiteSuppressionReason;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for manually suppressing an address */
model SiteCreateSuppressionInput {
  email: EmailAddress;
}

@route("/w/{workspaceSlug}/suppressions")
@tag("Site Suppressions")
namespace SiteSuppressions {
  /** Manually suppress an address (reason = manual; idempotent per address) */
  @createsResource(SiteSuppressionResource)
  @post
  op create(
    @path workspaceSlug: string,
    @body body: SiteCreateSuppressionInput,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: SiteSuppressionResource;
      }
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** List suppressed addresses */
  @listsResource(SiteSuppressionResource)
  @get
  op list(@path workspaceSlug: string, ...PageQuery):
    | Page<SiteSuppressionResource>
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Remove an address from the suppression list */
  @deletesResource(SiteSuppressionResource)
  @delete
  @route("/{id}")
  op delete(@path workspaceSlug: string, @path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;
}

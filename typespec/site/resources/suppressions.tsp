using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Why a destination is on the suppression (do-not-send) list. Unsubscribe is
 * not a suppression — it is a separate, per-source opt-out. */
enum SiteSuppressionReason {
  bounce,
  complaint,
  manual,
}

/** The channel a suppression applies to (email today; sms reserved) */
enum SiteSuppressionChannel {
  email,
}

/** A suppressed destination: the send path skips it on every surface */
@resource("suppressions")
@segment("suppressions")
model SiteSuppressionResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** The channel this suppression applies to */
  channel: SiteSuppressionChannel;

  /** Normalized (lower-cased) channel-specific destination address */
  destination: string;

  /** Why the destination is suppressed */
  reason: SiteSuppressionReason;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for manually suppressing a destination */
model SiteCreateSuppressionInput {
  /** Channel (defaults to email when omitted) */
  channel?: SiteSuppressionChannel;

  /** The destination address to suppress */
  destination: string;
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

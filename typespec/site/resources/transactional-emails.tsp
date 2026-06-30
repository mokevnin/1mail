using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Outcome of a transactional send */
enum SiteTransactionalEmailStatus {
  /** Claimed but not yet completed (in flight, or left behind by a crash) */
  pending,

  /** Accepted by the email provider */
  sent,

  /** Skipped: the destination was on the workspace suppression list */
  suppressed,

  /** The provider call failed */
  failed,
}

/** The channel a transactional send went out on (email today; sms reserved) */
enum SiteTransactionalEmailChannel {
  email,
}

/** One transactional send: the durable, read-only trace of an `/api/emails` call */
@resource("transactional-emails")
@segment("transactional-emails")
model SiteTransactionalEmailResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** The channel this send went out on */
  channel: SiteTransactionalEmailChannel;

  /** Normalized (lower-cased) destination address */
  destination: string;

  /** The id of the Template rendered at send time (bound by reference) */
  templateId: EntityId;

  /** The contact this destination resolved to, when one existed */
  contactId?: EntityId | null;

  /** Outcome of the send */
  status: SiteTransactionalEmailStatus;

  /** Provider error message, present when status is `failed` */
  error?: string | null;

  /** Creation timestamp */
  createdAt: Timestamp;
}

@route("/w/{workspaceSlug}/transactional-emails")
@tag("Site Transactional Emails")
namespace SiteTransactionalEmails {
  /** List transactional sends, most recent first */
  @listsResource(SiteTransactionalEmailResource)
  @get
  op list(@path workspaceSlug: string, ...PageQuery):
    | Page<SiteTransactionalEmailResource>
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;
}

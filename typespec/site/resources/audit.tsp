using TypeSpec.Http;

namespace OneMail;

/** Who performed an audited change */
enum SiteAuditActorKind {
  user,
  api_token,
  operator,
  system,
}

/** The actor of an Audit entry */
model SiteAuditActor {
  /** The kind of actor; an operator is shown to the customer as "1mail staff" */
  kind: SiteAuditActorKind;

  /** The actor's id (a User id or an API token id); absent for the system */
  id?: string | null;

  /** Display name snapshot taken when the entry was written */
  name?: string | null;
}

/** What an audited change was done to */
model SiteAuditTarget {
  /** Entity type, e.g. membership */
  type: string;

  /** The target's id */
  id?: string | null;

  /** Name snapshot taken when the entry was written; absent for a Contact */
  name?: string | null;
}

/** One immutable Audit entry (Enterprise) */
model SiteAuditEntryResource {
  /** Unique identifier */
  id: EntityId;

  /** When the change happened */
  occurredAt: Timestamp;

  /** Who did it */
  actor: SiteAuditActor;

  /** What was done, as `<entity>.<verb>` */
  action: string;

  /** What it was done to */
  target: SiteAuditTarget;

  /** Before/after of the changed fields; secrets appear as changed only */
  diff?: Record<unknown> | null;

  /** Request id of the change */
  requestId?: string | null;

  /** Client address of the change */
  ip?: string | null;

  /** Client user agent of the change */
  userAgent?: string | null;
}

/** A page of Audit entries, newest first */
model SiteAuditEntryList {
  items: SiteAuditEntryResource[];

  /** Pass as `cursor` for the next page; absent on the last page */
  nextCursor?: string | null;
}

@route("/workspaces/{slug}/audit-entries")
@tag("Site Audit")
namespace SiteAudit {
  /** List the workspace's Audit log, newest first (Enterprise; owner and admin only) */
  @get
  op list(
    @path slug: string,

    /** Cursor from a previous page */
    @query cursor?: string,

    /** Page size */
    @query limit?: int32 = 25,
  ):
    | SiteAuditEntryList
    | BadRequestProblem
    | PaymentRequiredProblem
    | ForbiddenProblem
    | NotFoundProblem;
}

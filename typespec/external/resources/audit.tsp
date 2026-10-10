using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OneMail;

/** Who performed an audited change */
enum AuditActorKind {
  user,
  api_token,
  operator,
  system,
}

/** The actor of an Audit entry */
model AuditActor {
  /** The kind of actor; an operator is shown as "1mail staff" */
  kind: AuditActorKind;

  /** The actor's id (a User id or an API token id); absent for the system */
  id?: string | null;

  /** Display name snapshot taken when the entry was written */
  name?: string | null;
}

/** What an audited change was done to */
model AuditTarget {
  /** Entity type, e.g. membership */
  type: string;

  /** The target's id */
  id?: string | null;

  /** Name snapshot taken when the entry was written; absent for a Contact */
  name?: string | null;
}

/** One immutable Audit entry (Enterprise) */
model AuditEntryResource {
  /** Unique identifier */
  id: EntityId;

  /** When the change happened */
  occurredAt: Timestamp;

  /** Who did it */
  actor: AuditActor;

  /** What was done, as `<entity>.<verb>` */
  action: string;

  /** What it was done to */
  target: AuditTarget;

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
model AuditEntryList {
  items: AuditEntryResource[];

  /** Pass as `cursor` for the next page; absent on the last page */
  nextCursor?: string | null;
}

@route("/audit-entries")
@tag("Audit")
namespace AuditEntries {
  /** List the Workspace's Audit log, newest first (Enterprise; needs the `audit:read` scope) */
  @get
  op list(
    /** Cursor from a previous page */
    @query cursor?: string,

    /** Page size */
    @query limit?: int32 = 25,
  ):
    | AuditEntryList
    | BadRequestProblem
    | UnauthorizedProblem
    | PaymentRequiredProblem;
}

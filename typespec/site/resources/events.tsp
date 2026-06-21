using TypeSpec.Http;

namespace OneMail;

/** A tracked event shown in the activity feed */
model SiteEventResource {
  /** Unique identifier */
  id: EntityId;

  /** Subject (tracking profile / visitor) the event belongs to */
  subjectId: string;

  /** Email associated with the event, if known */
  email?: string | null;

  /** Event action / name */
  action: string;

  /** Arbitrary event properties */
  properties?: Record<unknown> | null;

  /** When the event occurred at the source, if provided */
  occurredAt?: Timestamp | null;

  /** When the event was recorded */
  createdAt: Timestamp;
}

@route("/w/{workspaceSlug}/events")
@tag("Site Events")
namespace SiteEvents {
  /** List events for the site UI (most recent first) */
  @get
  op list(@path workspaceSlug: string, ...PageQuery):
    | Page<SiteEventResource>
    | NotFoundProblem;
}

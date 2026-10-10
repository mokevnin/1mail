using TypeSpec.Http;

namespace Sphericon;

/** Who set a Workspace suspension (ADR 0026) */
enum OperatorSuspensionActorKind {
  system,
  cli,
  operator,
}

/** The actor of a suspension. Unlike a customer, an Operator sees the real Operator id. */
model OperatorSuspensionActor {
  kind: OperatorSuspensionActorKind;

  /** The Operator id; present only for the `operator` kind */
  id?: string | null;
}

/** A Workspace's suspension state (ADR 0007) */
model OperatorSuspension {
  /** When outbound sending was suspended */
  at: Timestamp;

  /** Who suspended it */
  actor: OperatorSuspensionActor;

  /** Why, as given by the actor */
  reason?: string | null;
}

/** A Workspace as the Operator console sees it: metadata only, never Contacts,
 * content or Events (ADR 0026). */
model OperatorWorkspaceResource {
  /** Unique identifier */
  id: EntityId;

  slug: string;
  name: string;

  /** When the Workspace was created */
  createdAt: Timestamp;

  /** Present only while sending is suspended */
  suspension?: OperatorSuspension | null;
}

/** Search filter of the Workspace list */
model OperatorWorkspaceSearch {
  /** Only Workspaces whose slug contains this text (case-insensitive) */
  @query slug?: string;
}

@tag("Operator Workspaces")
@route("/workspaces")
namespace OperatorWorkspaces {
  /** List all Workspaces, newest first, optionally narrowed by a slug search */
  @get
  op list(...OperatorWorkspaceSearch, ...PageQuery):
    | Page<OperatorWorkspaceResource>
    | UnauthorizedProblem;

  /** One Workspace's metadata and suspension state */
  @route("/{workspaceId}")
  @get
  op get(@path workspaceId: EntityId):
    | OperatorWorkspaceResource
    | UnauthorizedProblem
    | NotFoundProblem;
}

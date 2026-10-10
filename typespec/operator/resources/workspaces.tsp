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

/** Why a Workspace is being suspended; the owner is told */
model OperatorSuspendInput {
  /** The reason, required: the Workspace owner is told why sending stopped */
  @minLength(1)
  reason: string;
}

/** The outcome of a suspend or unsuspend: the Workspace as it now stands, and whether
 * anything changed (suspending a suspended Workspace and unsuspending an active one
 * change nothing). */
model OperatorSuspensionChange {
  changed: boolean;
  workspace: OperatorWorkspaceResource;
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

  /** Suspend a Workspace's outbound sending (ADR 0007) with a required reason, as the
   * signed-in Operator. Idempotent: an already-suspended Workspace is left as it was. */
  @route("/{workspaceId}/suspend")
  @post
  op suspend(@path workspaceId: EntityId, @body body: OperatorSuspendInput):
    | OperatorSuspensionChange
    | UnauthorizedProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Lift a Workspace's suspension; held sends resume. Idempotent: a Workspace that is
   * not suspended is left as it was. */
  @route("/{workspaceId}/unsuspend")
  @post
  op unsuspend(@path workspaceId: EntityId):
    | OperatorSuspensionChange
    | UnauthorizedProblem
    | NotFoundProblem;
}

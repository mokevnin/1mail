using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Automation lifecycle status */
enum SiteAutomationStatus {
  draft,
  active,
}

/** Automation resource used by the site UI */
@resource("automations")
@segment("automations")
model SiteAutomationResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Automation name */
  name: string;

  /** Lifecycle status */
  status: SiteAutomationStatus;

  /** Event action that enrolls a contact (e.g. "contact.created", "email.opened") */
  triggerEvent: string;

  /** JSON array of steps: [{type:"email",subject,body},{type:"wait",seconds}] */
  definition: string;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for creating an automation */
model SiteCreateAutomationInput {
  name: string;
  triggerEvent: string;
  definition?: string;
}

/** Site request body for updating an automation */
model SiteUpdateAutomationInput {
  name?: string;
  triggerEvent?: string;
  definition?: string;
}

@route("/w/{workspaceSlug}/automations")
@tag("Site Automations")
namespace SiteAutomations {
  /** Create an automation */
  @createsResource(SiteAutomationResource)
  @post
  op create(
    @path workspaceSlug: string,
    @body body: SiteCreateAutomationInput,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: SiteAutomationResource;
      }
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** List automations */
  @listsResource(SiteAutomationResource)
  @get
  op list(@path workspaceSlug: string, ...PageQuery):
    | Page<SiteAutomationResource>
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Get an automation by ID */
  @readsResource(SiteAutomationResource)
  @get
  @route("/{id}")
  op get(@path workspaceSlug: string, @path id: EntityId):
    | SiteAutomationResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Update an automation */
  @updatesResource(SiteAutomationResource)
  @put
  @route("/{id}")
  op update(
    @path workspaceSlug: string,
    @path id: EntityId,
    @body body: SiteUpdateAutomationInput,
  ):
    | SiteAutomationResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Delete an automation */
  @deletesResource(SiteAutomationResource)
  @delete
  @route("/{id}")
  op delete(@path workspaceSlug: string, @path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;

  /** Activate an automation (starts enrolling contacts) */
  @post
  @route("/{id}/activate")
  op activate(@path workspaceSlug: string, @path id: EntityId):
    | SiteAutomationResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Deactivate an automation (stops new enrollments) */
  @post
  @route("/{id}/deactivate")
  op deactivate(@path workspaceSlug: string, @path id: EntityId):
    | SiteAutomationResource
    | BadRequestProblem
    | NotFoundProblem;
}

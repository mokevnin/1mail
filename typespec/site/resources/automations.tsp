using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

namespace OneMail;

/** Automation lifecycle status */
enum SiteAutomationStatus {
  draft,
  active,
}

/** Kind of automation step */
enum SiteAutomationStepType {
  email,
  wait,
}

/** One node in an automation's ordered, linear sequence. A flat shape: email
 * steps carry subject/body, wait steps carry seconds. */
model SiteAutomationStep {
  /** Step kind */
  type: SiteAutomationStepType;

  /** Email subject (email steps) */
  subject?: string;

  /** Email body as MJML (email steps) */
  body?: string;

  /** Delay before the next step, in seconds (wait steps) */
  seconds?: int32;
}

/** Automation resource used by the site UI */
@resource("automations")
@segment("automations")
@parentResource(SiteWorkspaceResource)
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

  /** The automation's ordered, linear steps */
  steps: SiteAutomationStep[];

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for creating an automation */
model SiteCreateAutomationInput {
  name: string;
  triggerEvent: string;
  steps?: SiteAutomationStep[];
}

/** Site request body for updating an automation */
model SiteUpdateAutomationInput {
  name?: string;
  triggerEvent?: string;
  steps?: SiteAutomationStep[];
}

@tag("Site Automations")
interface SiteAutomations
  extends WorkspaceCrud<
      SiteAutomationResource,
      SiteCreateAutomationInput,
      SiteUpdateAutomationInput
    > {
  /** Activate an automation (starts enrolling contacts) */
  @action("activate")
  @autoRoute
  activate(...ResourceParameters<SiteAutomationResource>):
    | SiteAutomationResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Deactivate an automation (stops new enrollments) */
  @action("deactivate")
  @autoRoute
  deactivate(...ResourceParameters<SiteAutomationResource>):
    | SiteAutomationResource
    | BadRequestProblem
    | NotFoundProblem;
}

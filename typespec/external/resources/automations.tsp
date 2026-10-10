using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

namespace OneMail;

/** Automation lifecycle status */
enum AutomationStatus {
  draft,
  active,
}

/** Kind of automation step */
enum AutomationStepType {
  email,
  wait,
  apply_tag,
  remove_tag,
}

/** One node in an automation's ordered, linear sequence. A flat shape: email
 * steps carry subject/body, wait steps carry seconds, tag steps carry the tag name. */
model AutomationStep {
  /** Step kind */
  type: AutomationStepType;

  /** Email subject (email steps) */
  subject?: string;

  /** Email body as MJML (email steps) */
  body?: string;

  /** Delay before the next step, in seconds (wait steps) */
  seconds?: int32;

  /** Tag name to apply to or remove from the enrolled contact (apply_tag and remove_tag steps) */
  tag?: string;
}

/** An Automation: an event-triggered, ordered sequence of steps. Created
 * automations are always draft; activating is a separate, send-class operation. */
@resource("automations")
@segment("automations")
model AutomationResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Automation name */
  name: string;

  /** Lifecycle status */
  status: AutomationStatus;

  /** Event action that enrolls a contact (e.g. "contact.created", "email.opened") */
  triggerEvent: string;

  /** The automation's ordered, linear steps */
  steps: AutomationStep[];

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating an automation (always created as a draft) */
model CreateAutomationInput {
  name: string;
  triggerEvent: string;
  steps?: AutomationStep[];
}

/** Request body for updating an automation */
model UpdateAutomationInput {
  name?: string;
  triggerEvent?: string;
  steps?: AutomationStep[];
}

@tag("Automations")
interface Automations
  extends ExternalCrud<
      AutomationResource,
      CreateAutomationInput,
      UpdateAutomationInput
    > {
  /** Activate an automation (starts enrolling contacts). Needs the automations:activate scope. */
  @action("activate")
  @autoRoute
  activate(...ResourceParameters<AutomationResource>):
    | AutomationResource
    | BadRequestProblem
    | UnauthorizedProblem
    | TooManyRequestsProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Deactivate an automation (stops new enrollments; in-flight ones finish). Needs the automations:activate scope. */
  @action("deactivate")
  @autoRoute
  deactivate(...ResourceParameters<AutomationResource>):
    | AutomationResource
    | BadRequestProblem
    | UnauthorizedProblem
    | TooManyRequestsProblem
    | NotFoundProblem;
}

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

namespace OneMail;

/** Broadcast lifecycle status */
enum SiteBroadcastStatus {
  draft,
  scheduled,
  sending,
  sent,
  failed,
}

/** Denormalized delivery counters for a broadcast */
model SiteBroadcastStats {
  /** Number of recipients the broadcast targets */
  recipientsTotal: int32;

  /** Messages successfully handed to the provider */
  sentCount: int32;

  /** Recipients that opened the email */
  openedCount: int32;

  /** Recipients that clicked a link */
  clickedCount: int32;

  /** Recipients that unsubscribed via this broadcast */
  unsubscribedCount: int32;

  /** Messages that failed to send */
  failedCount: int32;

  /** Messages handed to the provider ÷ recipients targeted (0–1) */
  deliveryRate: float32;

  /** Unique opens ÷ messages sent (0–1) */
  openRate: float32;

  /** Unique clicks ÷ messages sent (0–1) */
  clickRate: float32;

  /** Unique clicks ÷ unique opens (0–1) */
  clickToOpenRate: float32;

  /** Unsubscribes ÷ messages sent (0–1) */
  unsubscribeRate: float32;

  /** Send failures ÷ recipients targeted (0–1) */
  failureRate: float32;
}

/** Broadcast resource used by the site UI */
@resource("broadcasts")
@segment("broadcasts")
@parentResource(SiteWorkspaceResource)
model SiteBroadcastResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Internal broadcast name */
  name: string;

  /** Email subject line */
  subject: string;

  /** Sender display name (defaults to the integration's) */
  fromName?: string | null;

  /** Sender email address (defaults to the integration's) */
  fromEmail?: EmailAddress | null;

  /** MJML body authored in the composer */
  body: string;

  /** Derived plain-text body */
  bodyText: string;

  /** Target segment; null means all active contacts */
  segmentId?: EntityId | null;

  /** Sending integration; null means the workspace default */
  integrationId?: EntityId | null;

  /** Lifecycle status */
  status: SiteBroadcastStatus;

  /** When the broadcast is scheduled to send */
  scheduledAt?: Timestamp | null;

  /** When the broadcast finished sending */
  sentAt?: Timestamp | null;

  /** Delivery counters */
  stats: SiteBroadcastStats;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for creating a broadcast */
model SiteCreateBroadcastInput {
  /** Internal broadcast name */
  name: string;

  /** Email subject line */
  subject?: string;

  /** Sender display name */
  fromName?: string | null;

  /** Sender email address */
  fromEmail?: EmailAddress | null;

  /** MJML body */
  body?: string;

  /** Target segment; null/omitted means all active contacts */
  segmentId?: EntityId | null;

  /** Sending integration; null/omitted means the workspace default */
  integrationId?: EntityId | null;
}

/** Site request body for updating a broadcast */
model SiteUpdateBroadcastInput {
  /** Internal broadcast name */
  name?: string;

  /** Email subject line */
  subject?: string;

  /** Sender display name */
  fromName?: string | null;

  /** Sender email address */
  fromEmail?: EmailAddress | null;

  /** MJML body */
  body?: string;

  /** Target segment; null means all active contacts */
  segmentId?: EntityId | null;

  /** Sending integration; null means the workspace default */
  integrationId?: EntityId | null;
}

/** Site request body for scheduling a broadcast */
model SiteScheduleBroadcastInput {
  /** When the broadcast should be sent */
  scheduledAt: Timestamp;
}

/** Site request body for a test send */
model SiteTestSendBroadcastInput {
  /** Address to send the rendered preview to */
  email: EmailAddress;
}

@tag("Site Broadcasts")
interface SiteBroadcasts
  extends WorkspaceCrud<
      SiteBroadcastResource,
      SiteCreateBroadcastInput,
      SiteUpdateBroadcastInput
    > {
  /** Send a broadcast immediately */
  @action("send")
  @autoRoute
  send(...ResourceParameters<SiteBroadcastResource>):
    | SiteBroadcastResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Schedule a broadcast to send at a future time */
  @action("schedule")
  @autoRoute
  schedule(
    ...ResourceParameters<SiteBroadcastResource>,
    @body body: SiteScheduleBroadcastInput,
  ):
    | SiteBroadcastResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Send a one-off rendered preview to a single address */
  @action("test-send")
  @autoRoute
  testSend(
    ...ResourceParameters<SiteBroadcastResource>,
    @body body: SiteTestSendBroadcastInput,
  ):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;
}

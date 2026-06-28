using TypeSpec.Http;
using TypeSpec.Rest;

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
}

/** Broadcast resource used by the site UI */
@resource("broadcasts")
@segment("broadcasts")
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

@route("/w/{workspaceSlug}/broadcasts")
@tag("Site Broadcasts")
namespace SiteBroadcasts {
  /** Create a broadcast from the site UI */
  @createsResource(SiteBroadcastResource)
  @post
  op create(@path workspaceSlug: string, @body body: SiteCreateBroadcastInput):
    | {
        @statusCode statusCode: 201;
        @body body: SiteBroadcastResource;
      }
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** List broadcasts for the site UI */
  @listsResource(SiteBroadcastResource)
  @get
  op list(@path workspaceSlug: string, ...PageQuery):
    | Page<SiteBroadcastResource>
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Get a broadcast by ID for the site UI */
  @readsResource(SiteBroadcastResource)
  @get
  @route("/{id}")
  op get(@path workspaceSlug: string, @path id: EntityId):
    | SiteBroadcastResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Update a broadcast from the site UI */
  @updatesResource(SiteBroadcastResource)
  @put
  @route("/{id}")
  op update(
    @path workspaceSlug: string,
    @path id: EntityId,
    @body body: SiteUpdateBroadcastInput,
  ):
    | SiteBroadcastResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Delete a broadcast from the site UI */
  @deletesResource(SiteBroadcastResource)
  @delete
  @route("/{id}")
  op delete(@path workspaceSlug: string, @path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;

  /** Send a broadcast immediately */
  @post
  @route("/{id}/send")
  op send(@path workspaceSlug: string, @path id: EntityId):
    | SiteBroadcastResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Schedule a broadcast to send at a future time */
  @post
  @route("/{id}/schedule")
  op schedule(
    @path workspaceSlug: string,
    @path id: EntityId,
    @body body: SiteScheduleBroadcastInput,
  ):
    | SiteBroadcastResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Send a one-off rendered preview to a single address */
  @post
  @route("/{id}/test")
  op testSend(
    @path workspaceSlug: string,
    @path id: EntityId,
    @body body: SiteTestSendBroadcastInput,
  ):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;
}

using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Webhook endpoint resource used by the site UI */
@resource("webhooks")
@segment("webhooks")
model SiteWebhookEndpointResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Destination URL events are POSTed to */
  url: string;

  /** HMAC signing secret (used to verify the X-1mail-Signature header) */
  secret: string;

  /** Event names this endpoint receives; empty means all events */
  eventTypes: string[];

  /** Whether the endpoint receives deliveries */
  enabled: boolean;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for creating a webhook endpoint */
model SiteCreateWebhookEndpointInput {
  url: string;
  eventTypes?: string[];
  enabled?: boolean;
}

/** Site request body for updating a webhook endpoint */
model SiteUpdateWebhookEndpointInput {
  url?: string;
  eventTypes?: string[];
  enabled?: boolean;
}

@route("/w/{workspaceSlug}/webhooks")
@tag("Site Webhooks")
namespace SiteWebhooks {
  /** Create a webhook endpoint */
  @createsResource(SiteWebhookEndpointResource)
  @post
  op create(
    @path workspaceSlug: string,
    @body body: SiteCreateWebhookEndpointInput,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: SiteWebhookEndpointResource;
      }
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** List webhook endpoints */
  @listsResource(SiteWebhookEndpointResource)
  @get
  op list(@path workspaceSlug: string, ...PageQuery):
    | Page<SiteWebhookEndpointResource>
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Get a webhook endpoint by ID */
  @readsResource(SiteWebhookEndpointResource)
  @get
  @route("/{id}")
  op get(@path workspaceSlug: string, @path id: EntityId):
    | SiteWebhookEndpointResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Update a webhook endpoint */
  @updatesResource(SiteWebhookEndpointResource)
  @put
  @route("/{id}")
  op update(
    @path workspaceSlug: string,
    @path id: EntityId,
    @body body: SiteUpdateWebhookEndpointInput,
  ):
    | SiteWebhookEndpointResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Delete a webhook endpoint */
  @deletesResource(SiteWebhookEndpointResource)
  @delete
  @route("/{id}")
  op delete(@path workspaceSlug: string, @path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;
}

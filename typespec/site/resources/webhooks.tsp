using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Webhook endpoint resource used by the site UI */
@resource("webhooks")
@segment("webhooks")
@parentResource(SiteWorkspaceResource)
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

@tag("Site Webhooks")
interface SiteWebhooks
  extends WorkspaceCrud<
      SiteWebhookEndpointResource,
      SiteCreateWebhookEndpointInput,
      SiteUpdateWebhookEndpointInput
    > {}

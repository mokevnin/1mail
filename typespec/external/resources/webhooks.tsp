using TypeSpec.Http;
using TypeSpec.Rest;

namespace Sphericon;

/**
 * Outbound webhook endpoint. The HMAC signing secret is deliberately absent:
 * it is never returned through the external API (view it in the app).
 */
@resource("webhooks")
@segment("webhooks")
model WebhookResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Destination URL events are POSTed to */
  url: string;

  /** Event names this endpoint receives; empty means all events */
  eventTypes: string[];

  /** Whether the endpoint receives deliveries */
  enabled: boolean;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating a webhook endpoint */
model CreateWebhookInput {
  /** Destination URL (absolute http or https) */
  url: string;

  /** Event names to receive; empty or absent means all events */
  eventTypes?: string[];

  /** Whether the endpoint receives deliveries (default true) */
  enabled?: boolean;
}

/** Request body for updating a webhook endpoint */
model UpdateWebhookInput {
  /** Destination URL (absolute http or https) */
  url?: string;

  /** Event names to receive; empty means all events */
  eventTypes?: string[];

  /** Whether the endpoint receives deliveries */
  enabled?: boolean;
}

@tag("Webhooks")
interface Webhooks
  extends ExternalCrud<
      WebhookResource,
      CreateWebhookInput,
      UpdateWebhookInput
    > {}

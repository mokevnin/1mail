using TypeSpec.Http;

namespace OneMail;

/** Delivery channel an integration belongs to */
enum SiteIntegrationChannel {
  email,
  sms,
}

/** Concrete sending provider */
enum SiteIntegrationProvider {
  smtp,
  ses,
}

// --- Output config (secrets redacted; never echo passwords/keys) ---

/** SMTP config without the password */
model SiteSmtpConfig {
  kind: "smtp";
  host: string;
  port: int32;
  username?: string | null;
  from: EmailAddress;
  fromName?: string | null;
}

/** SES config without the secret access key */
model SiteSesConfig {
  kind: "ses";
  region: string;
  from: EmailAddress;
  fromName?: string | null;

  /** Last 4 chars of the access key id, for recognition */
  accessKeyIdLast4?: string | null;
}

/** Provider config returned to the UI, discriminated by `kind`. Secrets are omitted. */
@discriminated(#{ envelope: "none", discriminatorPropertyName: "kind" })
union SiteIntegrationConfig {
  smtp: SiteSmtpConfig,
  ses: SiteSesConfig,
}

// --- Input config (carries secrets; write-only) ---

/** SMTP credentials */
model SiteSmtpConfigInput {
  kind: "smtp";
  host: string;
  port: int32;
  username?: string | null;
  password?: string | null;
  from: EmailAddress;
  fromName?: string | null;
}

/** SES credentials */
model SiteSesConfigInput {
  kind: "ses";
  region: string;
  accessKeyId: string;
  secretAccessKey: string;
  from: EmailAddress;
  fromName?: string | null;
}

/** Provider config submitted by the UI, discriminated by `kind`. */
@discriminated(#{ envelope: "none", discriminatorPropertyName: "kind" })
union SiteIntegrationConfigInput {
  smtp: SiteSmtpConfigInput,
  ses: SiteSesConfigInput,
}

// --- Resource + request bodies ---

/** A workspace sending-provider integration */
model SiteIntegrationResource {
  /** Unique identifier */
  id: EntityId;

  /** Human-readable label */
  name: string;

  /** Delivery channel */
  channel: SiteIntegrationChannel;

  /** Concrete provider */
  provider: SiteIntegrationProvider;

  /** Whether this integration may be used for sending */
  enabled: boolean;

  /** Whether this is the default provider for its channel */
  isDefault: boolean;

  /** Provider config (secrets redacted) */
  config: SiteIntegrationConfig;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Create a workspace integration. The provider/channel are derived from `config.kind`. */
model SiteCreateIntegrationInput {
  name: string;
  enabled?: boolean;
  isDefault?: boolean;
  config: SiteIntegrationConfigInput;
}

/** Update a workspace integration. Omit `config` to keep stored credentials. */
model SiteUpdateIntegrationInput {
  name?: string;
  enabled?: boolean;
  isDefault?: boolean;
  config?: SiteIntegrationConfigInput | null;
}

@route("/w/{workspaceSlug}/integrations")
@tag("Site Integrations")
namespace SiteIntegrations {
  /** List the workspace's sending-provider integrations */
  @get
  op list(@path workspaceSlug: string):
    | {
        @statusCode statusCode: 200;
        @body body: SiteIntegrationResource[];
      }
    | NotFoundProblem;

  /** Create an integration */
  @post
  op create(@path workspaceSlug: string, @body body: SiteCreateIntegrationInput):
    | {
        @statusCode statusCode: 201;
        @body body: SiteIntegrationResource;
      }
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** Get an integration by id */
  @get
  @route("/{id}")
  op get(@path workspaceSlug: string, @path id: EntityId):
    | SiteIntegrationResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Update an integration */
  @put
  @route("/{id}")
  op update(
    @path workspaceSlug: string,
    @path id: EntityId,
    @body body: SiteUpdateIntegrationInput,
  ):
    | SiteIntegrationResource
    | BadRequestProblem
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** Delete an integration */
  @delete
  @route("/{id}")
  op delete(@path workspaceSlug: string, @path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;
}

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

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

  /** Custom SES-compatible API endpoint (e.g. Yandex Cloud Postbox); empty for AWS */
  endpoint?: string | null;

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

  /** Custom SES-compatible API endpoint (e.g. Yandex Cloud Postbox); empty for AWS */
  endpoint?: string | null;
}

/** Provider config submitted by the UI, discriminated by `kind`. */
@discriminated(#{ envelope: "none", discriminatorPropertyName: "kind" })
union SiteIntegrationConfigInput {
  smtp: SiteSmtpConfigInput,
  ses: SiteSesConfigInput,
}

// --- Resource + request bodies ---

/** A workspace sending-provider integration */
@resource("integrations")
@segment("integrations")
@parentResource(SiteWorkspaceResource)
model SiteIntegrationResource {
  /** Unique identifier */
  @key
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

// Bespoke (not WorkspaceCrud): `list` returns a bare array, not a paginated
// envelope. Nesting under /workspaces/{slug} comes from @parentResource + @autoRoute.
@tag("Site Integrations")
namespace SiteIntegrations {
  /** List the workspace's sending-provider integrations */
  @listsResource(SiteIntegrationResource)
  @get
  @autoRoute
  op list(...ResourceCollectionParameters<SiteIntegrationResource>):
    | {
        @statusCode statusCode: 200;
        @body body: SiteIntegrationResource[];
      }
    | NotFoundProblem;

  /** Create an integration */
  @createsResource(SiteIntegrationResource)
  @post
  @autoRoute
  op create(
    ...ResourceCollectionParameters<SiteIntegrationResource>,
    @body body: SiteCreateIntegrationInput,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: SiteIntegrationResource;
      }
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** Get an integration by id */
  @readsResource(SiteIntegrationResource)
  @get
  @autoRoute
  op get(...ResourceParameters<SiteIntegrationResource>):
    | SiteIntegrationResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Update an integration */
  @updatesResource(SiteIntegrationResource)
  @put
  @autoRoute
  op update(
    ...ResourceParameters<SiteIntegrationResource>,
    @body body: SiteUpdateIntegrationInput,
  ):
    | SiteIntegrationResource
    | BadRequestProblem
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** Delete an integration */
  @deletesResource(SiteIntegrationResource)
  @delete
  @autoRoute
  op delete(...ResourceParameters<SiteIntegrationResource>):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;
}

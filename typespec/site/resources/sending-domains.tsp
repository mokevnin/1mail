using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

namespace OneMail;

/** A DNS record the user publishes to authenticate a sending domain */
model SiteDnsRecord {
  /** The record type (always TXT for the records we generate) */
  type: "TXT";

  /** The host / name to create the record at */
  host: string;

  /** The record value to publish */
  value: string;
}

/**
 * Sending domain resource used by the site UI (ADR 0010). 1mail generates the
 * DKIM keypair; the user publishes the DKIM TXT to authenticate the domain. The
 * private key is never exposed. `verified` is a live property re-checked in the
 * background — it can flip back if the DNS record disappears.
 */
@resource("sending-domains")
@segment("sending-domains")
@parentResource(SiteWorkspaceResource)
model SiteSendingDomainResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** The authenticated domain, e.g. "mail.acme.com" */
  domain: string;

  /** DKIM selector; the record host is "<selector>._domainkey.<domain>" */
  dkimSelector: string;

  /** Whether the DKIM DNS is currently published and matches our key */
  verified: boolean;

  /** The required DKIM TXT record (gates sending in a later slice) */
  dkimRecord: SiteDnsRecord;

  /** The suggested SPF TXT record (advisory — does not gate) */
  spfRecord: SiteDnsRecord;

  /** The suggested DMARC TXT record (advisory bulk-readiness signal) */
  dmarcRecord: SiteDnsRecord;

  /** When the DKIM DNS was most recently checked (null = never) */
  lastCheckedAt?: Timestamp | null;

  /** When the domain most recently became verified (null = never) */
  verifiedAt?: Timestamp | null;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for adding a sending domain */
model SiteCreateSendingDomainInput {
  /** The domain to authenticate, e.g. "mail.acme.com" */
  domain: string;

  /** DKIM selector; defaults to "1mail" when omitted */
  dkimSelector?: string;
}

@tag("Site Sending Domains")
interface SiteSendingDomains {
  /** List sending domains for the site UI */
  @listsResource(SiteSendingDomainResource)
  @get
  @autoRoute
  list(
    ...ResourceCollectionParameters<SiteSendingDomainResource>,
    ...PageQuery,
  ):
    | Page<SiteSendingDomainResource>
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Add a sending domain; mints the DKIM keypair and returns the DNS records */
  @createsResource(SiteSendingDomainResource)
  @post
  @autoRoute
  create(
    ...ResourceCollectionParameters<SiteSendingDomainResource>,
    @body body: SiteCreateSendingDomainInput,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: SiteSendingDomainResource;
      }
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** Get a sending domain by ID for the site UI */
  @readsResource(SiteSendingDomainResource)
  @get
  @autoRoute
  get(...ResourceParameters<SiteSendingDomainResource>):
    | SiteSendingDomainResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Delete a sending domain */
  @deletesResource(SiteSendingDomainResource)
  @delete
  @autoRoute
  delete(...ResourceParameters<SiteSendingDomainResource>):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;

  /** Trigger an immediate DKIM re-check (fire-and-forget) */
  @action("verify")
  @autoRoute
  verify(...ResourceParameters<SiteSendingDomainResource>):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;
}

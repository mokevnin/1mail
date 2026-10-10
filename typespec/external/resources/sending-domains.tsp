using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;
using TypeSpec.OpenAPI;

namespace OneMail;

/** A DNS record the caller publishes to authenticate a Sending domain */
model DnsRecord {
  /** The record type (always TXT for the records we generate) */
  type: "TXT";

  /** The host / name to create the record at */
  host: string;

  /** The record value to publish */
  value: string;
}

/**
 * Sending domain (ADR 0010). 1mail generates the DKIM keypair; the caller publishes the
 * DKIM TXT record. The private key is never returned. `verified` is a live property
 * re-checked in the background: it can flip back if the DNS record disappears.
 */
@resource("sending-domains")
@segment("sending-domains")
model SendingDomainResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** The authenticated domain, e.g. "mail.acme.com" */
  domain: string;

  /** DKIM selector; the record host is "<selector>._domainkey.<domain>" */
  dkimSelector: string;

  /** Whether the DKIM DNS is currently published and matches our key */
  verified: boolean;

  /** The required DKIM TXT record */
  dkimRecord: DnsRecord;

  /** The suggested SPF TXT record (advisory: does not gate sending) */
  spfRecord: DnsRecord;

  /** The suggested DMARC TXT record (advisory bulk-readiness signal) */
  dmarcRecord: DnsRecord;

  /** When the DKIM DNS was most recently checked (null = never) */
  lastCheckedAt?: Timestamp | null;

  /** When the domain most recently became verified (null = never) */
  verifiedAt?: Timestamp | null;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Add a Sending domain; mints the DKIM keypair and returns the DNS records to publish. */
model CreateSendingDomainInput {
  /** The domain to authenticate, e.g. "mail.acme.com" (normalized to lowercase ASCII) */
  domain: string;

  /** DKIM selector; defaults to "1mail" when omitted */
  dkimSelector?: string;
}

/** Change a Sending domain. The domain name is immutable; changing the selector moves the DKIM record, so the domain becomes unverified until the new record is published. */
model UpdateSendingDomainInput {
  /** New DKIM selector */
  dkimSelector?: string;
}

@tag("Sending domains")
interface SendingDomains
  extends ExternalCrudUnique<
      SendingDomainResource,
      CreateSendingDomainInput,
      UpdateSendingDomainInput
    > {
  /** Trigger a live DKIM DNS check. The check runs in the background: the response is the domain as it stands now, 202 Accepted; read it again for the outcome (`verified`, `lastCheckedAt`). Verification is never set by the caller. */
  @action("verify")
  @autoRoute
  verify(...ResourceParameters<SendingDomainResource>):
    | {
        @statusCode statusCode: 202;
        @body body: SendingDomainResource;
      }
    | BadRequestProblem
    | UnauthorizedProblem
    | TooManyRequestsProblem
    | NotFoundProblem;
}

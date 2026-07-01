using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Contact resource used by the site UI */
@resource("contacts")
@segment("contacts")
@parentResource(SiteWorkspaceResource)
model SiteContactResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** The customer's own user id — an alias key. May be absent. */
  subjectId?: string | null;

  /** Email address — an alias key. May be absent for anonymous contacts. */
  email?: EmailAddress | null;

  /** Phone number — an alias key. May be absent. */
  phone?: string | null;

  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName | null;

  /** Typed custom field values keyed by the field's machine key */
  customFields?: Record<unknown> | null;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for creating a contact */
model SiteCreateContactInput {
  /** The customer's own user id — an alias key */
  subjectId?: string | null;

  /** Email address — optional; any alias key may be absent */
  email?: EmailAddress | null;

  /** Phone number — an alias key */
  phone?: string | null;

  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName | null;

  /** Typed custom field values keyed by the field's machine key */
  customFields?: Record<unknown> | null;
}

/** Site request body for updating a contact */
model SiteUpdateContactInput {
  /** The customer's own user id — an alias key */
  subjectId?: string | null;

  /** Email address — an alias key */
  email?: EmailAddress | null;

  /** Phone number — an alias key */
  phone?: string | null;

  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName | null;

  /** Typed custom field values keyed by the field's machine key */
  customFields?: Record<unknown> | null;
}

@tag("Site Contacts")
interface SiteContacts
  extends WorkspaceCrudUnique<
      SiteContactResource,
      SiteCreateContactInput,
      SiteUpdateContactInput
    > {}

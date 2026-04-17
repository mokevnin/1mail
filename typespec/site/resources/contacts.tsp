using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Contact status for site UI */
enum SiteContactStatus {
  active,
  unsubscribed,
}

/** Writable contact fields used by the site UI */
model SiteContactFields {
  /** Email address */
  email: EmailAddress;

  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName | null;

  /** Custom fields as key-value pairs */
  customFields?: Record<string> | null;
}

/** Updatable contact fields used by the site UI */
model SiteContactUpdatableFields {
  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName | null;

  /** Custom fields as key-value pairs */
  customFields?: Record<string> | null;
}

/** Contact resource used by the site UI */
@resource("contacts")
@segment("contacts")
model SiteContactResource extends SiteContactFields {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Current status */
  status: SiteContactStatus;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for creating a contact */
model SiteCreateContactInput extends SiteContactFields {}

/** Site request body for updating a contact */
model SiteUpdateContactInput extends SiteContactUpdatableFields {}

@route("/contacts")
@tag("Site Contacts")
namespace SiteContacts {
  /** Create a contact from the site UI */
  @createsResource(SiteContactResource)
  @post
  op create(@body body: SiteCreateContactInput):
    | {
        @statusCode statusCode: 201;
        @body body: SiteContactResource;
      }
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** List contacts for the site UI */
  @listsResource(SiteContactResource)
  @get
  op list(
    ...PageQuery,

    /** Filter by status */
    @query status?: SiteContactStatus,
  ):
    | Page<SiteContactResource>
    | BadRequestProblem
    | UnprocessableEntityProblem;

  /** Get a contact by ID for the site UI */
  @readsResource(SiteContactResource)
  @get
  @autoRoute
  op get(@path id: EntityId):
    | SiteContactResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Update a contact from the site UI */
  @updatesResource(SiteContactResource)
  @put
  @autoRoute
  op update(@path id: EntityId, @body body: SiteUpdateContactInput):
    | SiteContactResource
    | BadRequestProblem
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** Delete a contact from the site UI */
  @deletesResource(SiteContactResource)
  @delete
  @autoRoute
  op delete(@path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;
}

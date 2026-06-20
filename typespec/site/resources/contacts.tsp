using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Contact status for site UI */
enum SiteContactStatus {
  active,
  unsubscribed,
}

/** Contact resource used by the site UI */
@resource("contacts")
@segment("contacts")
model SiteContactResource {
  /** Unique identifier */
  @key
  id: EntityId;

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

  /** Current status */
  status: SiteContactStatus;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for creating a contact */
model SiteCreateContactInput {
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

/** Site request body for updating a contact */
model SiteUpdateContactInput {
  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName | null;

  /** Custom fields as key-value pairs */
  customFields?: Record<string> | null;
}

@route("/w/{workspaceSlug}/contacts")
@tag("Site Contacts")
namespace SiteContacts {
  /** Create a contact from the site UI */
  @createsResource(SiteContactResource)
  @post
  op create(@path workspaceSlug: string, @body body: SiteCreateContactInput):
    | {
        @statusCode statusCode: 201;
        @body body: SiteContactResource;
      }
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** List contacts for the site UI */
  @listsResource(SiteContactResource)
  @get
  op list(
    @path workspaceSlug: string,
    ...PageQuery,

    /** Filter by status */
    @query status?: SiteContactStatus,
  ):
    | Page<SiteContactResource>
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Get a contact by ID for the site UI */
  @readsResource(SiteContactResource)
  @get
  @route("/{id}")
  op get(@path workspaceSlug: string, @path id: EntityId):
    | SiteContactResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Update a contact from the site UI */
  @updatesResource(SiteContactResource)
  @put
  @route("/{id}")
  op update(
    @path workspaceSlug: string,
    @path id: EntityId,
    @body body: SiteUpdateContactInput,
  ):
    | SiteContactResource
    | BadRequestProblem
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** Delete a contact from the site UI */
  @deletesResource(SiteContactResource)
  @delete
  @route("/{id}")
  op delete(@path workspaceSlug: string, @path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;
}

using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Contact status */
enum ContactStatus {
  active,
  unsubscribed,
}

/** Contact resource */
@resource("contacts")
@segment("contacts")
model ContactResource {
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
  status: ContactStatus;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating a contact */
model CreateContactInput {
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

/** Request body for updating a contact */
model UpdateContactInput {
  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName | null;

  /** Custom fields as key-value pairs */
  customFields?: Record<string> | null;
}

@route("/contacts")
@tag("Contacts")
namespace Contacts {
  /** Create a contact */
  @createsResource(ContactResource)
  @post
  op create(@body body: CreateContactInput):
    | {
        @statusCode statusCode: 201;
        @body body: ContactResource;
      }
    | UnauthorizedProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** List contacts with pagination */
  @listsResource(ContactResource)
  @get
  op list(
    ...PageQuery,

    /** Filter by status */
    @query status?: ContactStatus,
  ):
    | Page<ContactResource>
    | BadRequestProblem
    | UnauthorizedProblem
    | UnprocessableEntityProblem;

  /** Get a contact by ID */
  @readsResource(ContactResource)
  @get
  @autoRoute
  op get(@path id: EntityId):
    | ContactResource
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem;

  /** Update a contact */
  @updatesResource(ContactResource)
  @put
  @autoRoute
  op update(@path id: EntityId, @body body: UpdateContactInput):
    | ContactResource
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** Delete a contact */
  @deletesResource(ContactResource)
  @delete
  @autoRoute
  op delete(@path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem;
}

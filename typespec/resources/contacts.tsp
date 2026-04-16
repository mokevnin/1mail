using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Contact status */
enum ContactStatus {
  active,
  unsubscribed,
}

/** Shared writable fields for contact */
model ContactFields {
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

/** Shared updatable fields for contact */
model ContactUpdatableFields {
  /** First name */
  firstName?: string | null;

  /** Last name */
  lastName?: string | null;

  /** IANA time zone identifier */
  timeZone?: TimeZoneName | null;

  /** Custom fields as key-value pairs */
  customFields?: Record<string> | null;
}

/** Contact resource */
@resource("contacts")
@segment("contacts")
model ContactResource extends ContactFields {
  /** Unique identifier */
  @key
  id: int64;

  /** Current status */
  status: ContactStatus;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Request body for creating a contact */
model CreateContactInput extends ContactFields {}

/** Request body for updating a contact */
model UpdateContactInput extends ContactUpdatableFields {}

model ContactPage is Page<ContactResource>;

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
    | ContactPage
    | BadRequestProblem
    | UnauthorizedProblem
    | UnprocessableEntityProblem;

  /** Get a contact by ID */
  @readsResource(ContactResource)
  @get
  @autoRoute
  op get(@path id: int64):
    | ContactResource
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem;

  /** Update a contact */
  @updatesResource(ContactResource)
  @put
  @autoRoute
  op update(@path id: int64, @body body: UpdateContactInput):
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
  op delete(@path id: int64):
    | NoContentResponse
    | BadRequestProblem
    | UnauthorizedProblem
    | NotFoundProblem;
}

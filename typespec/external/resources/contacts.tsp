using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Contact resource */
@resource("contacts")
@segment("contacts")
model ContactResource {
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

/** Request body for creating a contact */
model CreateContactInput {
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

/** Request body for updating a contact */
model UpdateContactInput {
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
  op list(...PageQuery):
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

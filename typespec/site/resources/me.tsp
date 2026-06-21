using TypeSpec.Http;

namespace OneMail;

/** The authenticated dashboard user */
model SiteUserResource {
  /** Unique identifier */
  id: EntityId;

  /** Display name */
  name: string;

  /** Email address (login identity; not editable here) */
  email: EmailAddress;

  /** Creation timestamp */
  createdAt: Timestamp;
}

/** Update the authenticated user's profile. To change the password, provide
 * both currentPassword and newPassword. */
model SiteUpdateMeInput {
  /** New display name */
  name?: string;

  /** Current password — required when changing the password */
  currentPassword?: string;

  /** New password */
  newPassword?: string;
}

@route("/me")
@tag("Site User")
namespace SiteUser {
  /** Get the authenticated user's profile */
  @get
  op getMe(): {
    @statusCode statusCode: 200;
    @body body: SiteUserResource;
  };

  /** Update the authenticated user's profile (name and/or password) */
  @put
  op updateMe(@body body: SiteUpdateMeInput):
    | SiteUserResource
    | ForbiddenProblem
    | UnprocessableEntityProblem;
}

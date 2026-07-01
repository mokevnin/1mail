using TypeSpec.Http;

namespace OneMail;

model SiteRegisterInput {
  name: string;
  email: EmailAddress;
  password: string;
}

model SiteRegisterResult {
  id: EntityId;
  name: string;
  email: EmailAddress;
  createdAt: Timestamp;
}

model SiteDirectLoginInput {
  user: string;
  passwd: string;
}

model SiteDirectLoginResult {
  name: string;
  id: string;
  picture?: string;
  aud?: string;
  ip?: string;
  email?: string;
  attrs?: Record<unknown>;
  role?: string;
}

model SiteDirectLoginError {
  error: string;
}

@error
model SiteDirectLoginForbidden is Response<403> {
  @header contentType: "application/json";
  @body body: SiteDirectLoginError;
}

/** Request a password-reset link (always accepted; no account enumeration) */
model SiteForgotPasswordInput {
  email: EmailAddress;
}

/** Set a new password using a reset token */
model SiteResetPasswordInput {
  token: string;
  password: string;
}

/** Confirm ownership of an email address using a verification token */
model SiteVerifyEmailInput {
  token: string;
}

/** Confirm a requested email change using the token sent to the new address */
model SiteConfirmEmailChangeInput {
  token: string;
}

@route("/auth")
@tag("Site Auth")
namespace SiteAuth {
  @useAuth(NoAuth)
  @post
  @route("/register")
  op register(@body body: SiteRegisterInput):
    | {
        @statusCode statusCode: 201;
        @body body: SiteRegisterResult;
      }
    | ConflictProblem
    | UnprocessableEntityProblem;

  @useAuth(NoAuth)
  @post
  @route("/direct/login")
  op directLogin(@body body: SiteDirectLoginInput):
    | {
        @statusCode statusCode: 200;
        @body body: SiteDirectLoginResult;
      }
    | SiteDirectLoginForbidden
    | BadRequestProblem;

  /** Request a password-reset link. Always returns 202 regardless of whether
   * the email matches an account, to avoid leaking which addresses exist. */
  @useAuth(NoAuth)
  @post
  @route("/forgot-password")
  op forgotPassword(@body body: SiteForgotPasswordInput): AcceptedResponse;

  /** Set a new password from a reset token */
  @useAuth(NoAuth)
  @post
  @route("/reset-password")
  op resetPassword(@body body: SiteResetPasswordInput):
    | OkResponse
    | BadRequestProblem;

  /** Confirm an email address from a verification token (signup verification) */
  @useAuth(NoAuth)
  @post
  @route("/verify-email")
  op verifyEmail(@body body: SiteVerifyEmailInput):
    | OkResponse
    | BadRequestProblem;

  /** Confirm an email change from the token sent to the new address. Public:
   * the link is opened from the new inbox, which has no session. */
  @useAuth(NoAuth)
  @post
  @route("/confirm-email-change")
  op confirmEmailChange(@body body: SiteConfirmEmailChangeInput):
    | OkResponse
    | BadRequestProblem
    | ConflictProblem;
}

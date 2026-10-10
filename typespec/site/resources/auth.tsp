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

/** Sign in with an email and password. The email is a plain string: a malformed
 * address must answer like an unknown one. */
model SiteLoginInput {
  email: string;
  password: string;
}

/** What a login granted. `session`: the session cookie is set. A User with a
 * Second factor will get a challenge outcome instead (ADR 0020). */
enum SiteLoginOutcome {
  session,
}

model SiteLoginResult {
  outcome: SiteLoginOutcome;
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
    | UnprocessableEntityProblem
    | TooManyRequestsProblem;

  /** Check the password and start a session (the JWT cookie). Unknown email and
   * wrong password answer the same 401; failures feed the Login throttle, which
   * answers 429 even for a correct password while its delay runs (ADR 0025). */
  @useAuth(NoAuth)
  @post
  @route("/login")
  op login(@body body: SiteLoginInput):
    | {
        @statusCode statusCode: 200;

        /** The session cookie, set when the outcome is `session` */
        @header("Set-Cookie") setCookie?: string;

        @body body: SiteLoginResult;
      }
    | UnauthorizedProblem
    | TooManyRequestsProblem;

  /** End the session on this browser: clears the session cookie. */
  @useAuth(NoAuth)
  @post
  @route("/logout")
  op logout(): {
    @statusCode statusCode: 204;

    /** The expired session cookie */
    @header("Set-Cookie") setCookie: string;
  };

  /** Request a password-reset link. Always returns 202 regardless of whether
   * the email matches an account and even when the address has used its hourly
   * mail budget, to avoid leaking which addresses exist. Answers 429 over the
   * per-IP limit. */
  @useAuth(NoAuth)
  @post
  @route("/forgot-password")
  op forgotPassword(@body body: SiteForgotPasswordInput):
    | AcceptedResponse
    | TooManyRequestsProblem;

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
   * the link may be opened from a browser without a session. The change ends
   * every session of the user (ADR 0020); when the request carries a valid
   * session of that same user, it continues under the fresh cookie set in this
   * response. The link alone never starts a session. */
  @useAuth(NoAuth)
  @post
  @route("/confirm-email-change")
  op confirmEmailChange(
    /** The confirming browser's session, if any */
    @cookie("JWT") session?: string,

    @body body: SiteConfirmEmailChangeInput,
  ):
    | {
        @statusCode statusCode: 200;

        /** The confirming session reissued, set when the request carried one */
        @header("Set-Cookie") setCookie?: string;
      }
    | BadRequestProblem
    | ConflictProblem;
}

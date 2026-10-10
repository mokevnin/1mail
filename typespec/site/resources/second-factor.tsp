using TypeSpec.Http;

namespace OneMail;

/** The authenticated User's Second factor (ADR 0020). Recovery codes are never
 * readable here: only how many are left. */
model SiteSecondFactorStatus {
  /** Whether a confirmed TOTP Second factor is active */
  enabled: boolean;

  /** Whether an enrollment was started and still awaits its confirmation code */
  pending: boolean;

  /** Unused Recovery codes left (0 without a Second factor) */
  recoveryCodesRemaining: int32;
}

/** A pending enrollment: the TOTP secret to add to an authenticator app */
model SiteSecondFactorEnrollment {
  /** The secret as a base32 key, for typing it into the app */
  secret: string;

  /** The otpauth:// URI the QR code encodes */
  otpauthUri: string;

  /** The QR code of the otpauth URI, as a PNG data URI */
  qrCode: string;
}

/** A code from the authenticator app */
model SiteSecondFactorConfirmInput {
  /** The current 6-digit TOTP code */
  code: string;
}

/** A fresh set of Recovery codes. They are shown only in this response. */
model SiteRecoveryCodes {
  /** Single-use codes, each one usable in place of a TOTP code */
  codes: string[];
}

/** Proof of the User's password, for regenerating Recovery codes */
model SiteRecoveryCodesInput {
  /** Current password */
  currentPassword: string;
}

/** Proof of password and possession, for disabling the Second factor */
model SiteSecondFactorDisableInput {
  /** Current password */
  currentPassword: string;

  /** A current TOTP code, or an unused Recovery code */
  code: string;
}

@route("/me/second-factor")
@tag("Site Second Factor")
namespace SiteSecondFactor {
  /** The authenticated User's Second factor status */
  @get
  op getStatus(): {
    @statusCode statusCode: 200;
    @body body: SiteSecondFactorStatus;
  };

  /** Start enrolling a TOTP Second factor: creates a pending secret (replacing an
   * earlier pending one). It counts as a Second factor only once confirmed. 409
   * when a Second factor is already active. */
  @post
  @route("/enrollment")
  op startEnrollment():
    | {
        @statusCode statusCode: 200;
        @body body: SiteSecondFactorEnrollment;
      }
    | ConflictProblem;

  /** Confirm the pending enrollment with a code from the app. On success the Second
   * factor is active, every other session ends and the acting one continues under
   * the cookie set here; the Recovery codes are returned once. 422 on a wrong code,
   * 409 without a pending enrollment. */
  @post
  @route("/enrollment/confirm")
  op confirmEnrollment(@body body: SiteSecondFactorConfirmInput):
    | {
        @statusCode statusCode: 200;

        /** The acting session reissued under the new session epoch */
        @header("Set-Cookie") setCookie: string;

        @body body: SiteRecoveryCodes;
      }
    | ConflictProblem
    | UnprocessableEntityProblem;

  /** Replace the Recovery codes with a fresh set (the previous set stops working).
   * Every other session ends; the acting one continues under the cookie set here.
   * 403 on a wrong password, 409 without an active Second factor. */
  @post
  @route("/recovery-codes")
  op regenerateRecoveryCodes(@body body: SiteRecoveryCodesInput):
    | {
        @statusCode statusCode: 200;

        /** The acting session reissued under the new session epoch */
        @header("Set-Cookie") setCookie: string;

        @body body: SiteRecoveryCodes;
      }
    | ForbiddenProblem
    | ConflictProblem;

  /** Disable the Second factor, proving the password and a current code. Every
   * other session ends; the acting one continues under the cookie set here. 403 on
   * a wrong password, 422 on a wrong code, 409 without an active Second factor. */
  @post
  @route("/disable")
  op disable(@body body: SiteSecondFactorDisableInput):
    | {
        @statusCode statusCode: 204;

        /** The acting session reissued under the new session epoch */
        @header("Set-Cookie") setCookie: string;
      }
    | ForbiddenProblem
    | ConflictProblem
    | UnprocessableEntityProblem;
}

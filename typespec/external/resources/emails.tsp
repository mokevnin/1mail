using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Outcome of a transactional send */
enum TransactionalSendStatus {
  /** Accepted by the email provider */
  sent,

  /** Skipped: the destination is on the workspace suppression list */
  suppressed,
}

/** Send a single transactional email by rendering a referenced template. */
model SendTransactionalEmailInput {
  /** The Template to render at send time (bound by reference, not copied) */
  templateId: EntityId;

  /** The destination email address */
  destination: EmailAddress;

  /** Per-call variables merged into the template's subject and body (Liquid `{{ }}`) */
  variables?: Record<unknown>;
}

/** Result of a transactional send */
model SendTransactionalEmailResponse {
  /** The id of the durable send record (the send-history trace) */
  id: EntityId;

  /** Whether the message was sent or skipped as suppressed */
  status: TransactionalSendStatus;

  /** The normalized destination the result applies to */
  destination: string;
}

@route("/emails")
@tag("Emails")
namespace Emails {
  /**
   * Send a transactional email. The referenced Template's current content is
   * rendered with the supplied variables at send time. Respects the workspace
   * Suppression list (a suppressed destination returns status `suppressed`) but
   * not marketing Unsubscribe — transactional mail carries no sending source.
   *
   * Supply an `Idempotency-Key` to make retries safe: a repeated key replays the
   * original result without sending a second email, and a key whose first request
   * is still in flight returns 409.
   */
  @post
  op send(
    @body body: SendTransactionalEmailInput,

    /** Client-supplied key that makes a retried send idempotent (Stripe-style) */
    @header("Idempotency-Key") idempotencyKey?: string,
  ):
    | {
        @statusCode statusCode: 202;
        @body body: SendTransactionalEmailResponse;
      }
    | UnauthorizedProblem
    | NotFoundProblem
    | ConflictProblem
    | UnprocessableEntityProblem;
}

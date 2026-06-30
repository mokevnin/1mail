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
   */
  @post
  op send(@body body: SendTransactionalEmailInput):
    | {
        @statusCode statusCode: 202;
        @body body: SendTransactionalEmailResponse;
      }
    | UnauthorizedProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;
}

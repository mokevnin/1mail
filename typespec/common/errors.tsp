using TypeSpec.Http;

namespace OneMail;

alias ValidationErrors = Record<string[]>;
alias FormValidationFields = Record<string>;

/** RFC 7807 Problem Details */
model ProblemDetails {
  /** A URI reference that identifies the problem type */
  type?: string;

  /** A short, human-readable summary of the problem type */
  title?: string;

  /** HTTP status code */
  status?: int32;

  /** A human-readable explanation specific to this occurrence */
  detail?: string;

  /** A URI reference that identifies the specific occurrence */
  instance?: string;

  /** Validation errors grouped by field */
  errors?: ValidationErrors;

  /** Form-level validation error */
  form?: string;

  /** Field validation errors */
  fields?: FormValidationFields;

  /** Seconds to wait before retrying; set on a 429 so a client that only sees the body can show the wait */
  retryAfter?: int32;
}

/** RFC 7807 bad request response */
@error
model BadRequestProblem is Response<400> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 gone response: the resource existed but is no longer available (an expired link) */
@error
model GoneProblem is Response<410> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 unauthorized response */
@error
model UnauthorizedProblem is Response<401> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 conflict response */
@error
model ConflictProblem is Response<409> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 not found response */
@error
model NotFoundProblem is Response<404> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 payment required response: the feature needs an Enterprise license */
@error
model PaymentRequiredProblem is Response<402> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 forbidden response */
@error
model ForbiddenProblem is Response<403> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

/** RFC 7807 too many requests response: a rate limit was exceeded (ADR 0025) */
@error
model TooManyRequestsProblem is Response<429> {
  @header contentType: "application/problem+json";

  /** Seconds to wait before retrying */
  @header("Retry-After") retryAfter: int32;

  /** The request budget of the window that was exceeded */
  @header("X-RateLimit-Limit") rateLimitLimit: int32;

  /** Requests left in the current window */
  @header("X-RateLimit-Remaining") rateLimitRemaining: int32;

  /** Unix time at which the window resets */
  @header("X-RateLimit-Reset") rateLimitReset: int64;

  @body body: ProblemDetails;
}

/** RFC 7807 validation response */
@error
model UnprocessableEntityProblem is Response<422> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

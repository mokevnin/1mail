using TypeSpec.Http;

namespace OneMail;

alias ValidationErrors = Record<string[]>;

/** RFC 7807 Problem Details */
@example(#{
  type: "https://onemail.dev/problems/validation-error",
  title: "Validation error",
  status: 422,
  detail: "Request validation failed",
  instance: "/contacts",
  errors: #{
    email: #["must be a well-formed email address"],
  },
})
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
}

/** RFC 7807 bad request response */
@error
model BadRequestProblem is Response<400> {
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

/** RFC 7807 validation response */
@error
model UnprocessableEntityProblem is Response<422> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

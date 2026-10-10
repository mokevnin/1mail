using TypeSpec.Http;

namespace Sphericon;

/** RFC 7807 payload too large response: the body exceeds the size cap (ADR 0025) */
@error
model PayloadTooLargeProblem is Response<413> {
  @header contentType: "application/problem+json";
  @body body: ProblemDetails;
}

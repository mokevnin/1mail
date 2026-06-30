using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** A Custom field's value type */
enum SiteCustomFieldType {
  string,
  number,
  bool,
  datetime,
}

/** Custom field definition — a typed, named Contact attribute (ADR 0006). Auto-created
 on first sight from Identify; the catalogue feeds the segment builder. */
@resource("custom-fields")
@segment("custom-fields")
model SiteCustomFieldResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Machine key — how values are addressed in custom fields and segment rules */
  key: string;

  /** Display name (renameable) */
  name: string;

  /** Inferred value type */
  type: SiteCustomFieldType;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

@route("/w/{workspaceSlug}/custom-fields")
@tag("Site Custom Fields")
namespace SiteCustomFields {
  /** List the workspace's custom field definitions */
  @listsResource(SiteCustomFieldResource)
  @get
  op list(@path workspaceSlug: string):
    | Page<SiteCustomFieldResource>
    | BadRequestProblem
    | NotFoundProblem;
}

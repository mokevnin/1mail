using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Workspace the current user belongs to */
@resource("workspaces")
@segment("workspaces")
model SiteWorkspaceResource {
  /** Unique identifier */
  id: EntityId;

  /** Display name */
  name: string;

  /** URL-safe unique slug; the route key for nested workspace resources */
  @key("slug")
  slug: string;

  /** Per-workspace collect write-key (x-collect-key) for the tracking snippet */
  collectKey: string;

  /** Secret key that routes inbound provider webhooks (/hooks/{ingestKey}/...) */
  ingestKey: string;

  /** Physical postal address printed in the marketing email footer (CAN-SPAM
   * 15 U.S.C. §7704(a)(5)). Empty until the workspace sets one. */
  postalAddress: string;

  /** Creation timestamp */
  createdAt: Timestamp;
}

/** Update a workspace. The slug is immutable (it is the route key and globally
 * unique); the display name and the CAN-SPAM postal address can change. */
model SiteUpdateWorkspaceInput {
  /** New display name */
  name: string;

  /** Physical postal address for the marketing email footer (CAN-SPAM). Omit to
   * leave it unchanged; send an empty string to clear it. */
  postalAddress?: string;
}

@route("/workspaces")
@tag("Site Workspaces")
namespace SiteWorkspaces {
  /** List workspaces owned by the authenticated user */
  @get
  op list(): {
    @statusCode statusCode: 200;
    @body body: SiteWorkspaceResource[];
  };

  /** Rename a workspace owned by the authenticated user */
  @put
  @route("/{slug}")
  op update(@path slug: string, @body body: SiteUpdateWorkspaceInput):
    | SiteWorkspaceResource
    | NotFoundProblem
    | UnprocessableEntityProblem;
}

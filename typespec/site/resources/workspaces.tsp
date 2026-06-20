using TypeSpec.Http;

namespace OneMail;

/** Workspace the current user belongs to */
model SiteWorkspaceResource {
  /** Unique identifier */
  id: EntityId;

  /** Display name */
  name: string;

  /** URL-safe unique slug (used in /w/{slug} routes) */
  slug: string;

  /** Per-workspace collect write-key (x-collect-key) for the tracking snippet */
  collectKey: string;

  /** Creation timestamp */
  createdAt: Timestamp;
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
}

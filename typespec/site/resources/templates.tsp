using TypeSpec.Http;
using TypeSpec.Rest;

namespace OneMail;

/** Authoring format of an email body */
enum SiteEmailBodyFormat {
  html,
  mjml,
}

/** Reusable email template used by the site UI */
@resource("templates")
@segment("templates")
model SiteEmailTemplateResource {
  /** Unique identifier */
  @key
  id: EntityId;

  /** Template name */
  name: string;

  /** Default subject line */
  subject: string;

  /** Body authoring format */
  bodyFormat: SiteEmailBodyFormat;

  /** Body (HTML or MJML depending on bodyFormat) */
  bodyHtml: string;

  /** Creation timestamp */
  createdAt: Timestamp;

  /** Last update timestamp */
  updatedAt: Timestamp;
}

/** Site request body for creating a template */
model SiteCreateEmailTemplateInput {
  name: string;
  subject?: string;
  bodyFormat?: SiteEmailBodyFormat;
  bodyHtml?: string;
}

/** Site request body for updating a template */
model SiteUpdateEmailTemplateInput {
  name?: string;
  subject?: string;
  bodyFormat?: SiteEmailBodyFormat;
  bodyHtml?: string;
}

@route("/w/{workspaceSlug}/templates")
@tag("Site Templates")
namespace SiteTemplates {
  /** Create a template */
  @createsResource(SiteEmailTemplateResource)
  @post
  op create(
    @path workspaceSlug: string,
    @body body: SiteCreateEmailTemplateInput,
  ):
    | {
        @statusCode statusCode: 201;
        @body body: SiteEmailTemplateResource;
      }
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** List templates */
  @listsResource(SiteEmailTemplateResource)
  @get
  op list(@path workspaceSlug: string, ...PageQuery):
    | Page<SiteEmailTemplateResource>
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Get a template by ID */
  @readsResource(SiteEmailTemplateResource)
  @get
  @route("/{id}")
  op get(@path workspaceSlug: string, @path id: EntityId):
    | SiteEmailTemplateResource
    | BadRequestProblem
    | NotFoundProblem;

  /** Update a template */
  @updatesResource(SiteEmailTemplateResource)
  @put
  @route("/{id}")
  op update(
    @path workspaceSlug: string,
    @path id: EntityId,
    @body body: SiteUpdateEmailTemplateInput,
  ):
    | SiteEmailTemplateResource
    | BadRequestProblem
    | NotFoundProblem
    | UnprocessableEntityProblem;

  /** Delete a template */
  @deletesResource(SiteEmailTemplateResource)
  @delete
  @route("/{id}")
  op delete(@path workspaceSlug: string, @path id: EntityId):
    | NoContentResponse
    | BadRequestProblem
    | NotFoundProblem;
}

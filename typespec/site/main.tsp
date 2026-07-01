import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";

import "../common/scalars.tsp";
import "../common/pagination.tsp";
import "../common/errors.tsp";
import "../common/crud.tsp";
import "./resources/contacts.tsp";
import "./resources/custom-fields.tsp";
import "./resources/segments.tsp";
import "./resources/auth.tsp";
import "./resources/workspaces.tsp";
import "./resources/me.tsp";
import "./resources/events.tsp";
import "./resources/tokens.tsp";
import "./resources/integrations.tsp";
import "./resources/broadcasts.tsp";
import "./resources/templates.tsp";
import "./resources/automations.tsp";
import "./resources/webhooks.tsp";
import "./resources/analytics.tsp";
import "./resources/suppressions.tsp";
import "./resources/transactional-emails.tsp";

using TypeSpec.OpenAPI;
using TypeSpec.Http;

@service(#{ title: "OneMail Site API" })
@info(#{ version: "1.0.0" })
@server("http://localhost:3000", "Local development site API")
@useAuth(ApiKeyAuth<ApiKeyLocation.cookie, "JWT">)
namespace OneMail;

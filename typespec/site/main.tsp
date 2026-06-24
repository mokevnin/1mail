import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";

import "../common/scalars.tsp";
import "../common/pagination.tsp";
import "../common/errors.tsp";
import "./resources/contacts.tsp";
import "./resources/segments.tsp";
import "./resources/auth.tsp";
import "./resources/workspaces.tsp";
import "./resources/me.tsp";
import "./resources/events.tsp";
import "./resources/tokens.tsp";

using TypeSpec.OpenAPI;
using TypeSpec.Http;

@service(#{ title: "OneMail Site API" })
@info(#{ version: "1.0.0" })
@server("http://localhost:3000", "Local development site API")
@useAuth(ApiKeyAuth<ApiKeyLocation.cookie, "JWT">)
namespace OneMail;

import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/versioning";

import "../common/scalars.tsp";
import "../common/pagination.tsp";
import "../common/errors.tsp";
import "../common/versions.tsp";
import "../common/crud.tsp";
import "./resources/contacts.tsp";
import "./resources/segments.tsp";
import "./resources/broadcasts.tsp";
import "./resources/emails.tsp";
import "./resources/events.tsp";
import "./resources/auth.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;
using TypeSpec.Versioning;

@service(#{ title: "OneMail External API" })
@info(#{ version: "1.0.0" })
@server("https://api.onemail.dev", "Production API")
@server("http://localhost:3000/api", "Local development API")
@useAuth(BearerAuth)
@versioned(Versions)
namespace OneMail;

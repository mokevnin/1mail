import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";

import "../common/scalars.tsp";
import "../common/pagination.tsp";
import "../common/errors.tsp";
import "./resources/contacts.tsp";

using TypeSpec.OpenAPI;
using TypeSpec.Http;

@service(#{ title: "OneMail Site API" })
@info(#{ version: "1.0.0" })
@server("http://localhost:3000", "Local development site API")
namespace OneMail;

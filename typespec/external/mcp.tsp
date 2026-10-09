import "@typespec/openapi";

using TypeSpec.OpenAPI;

namespace OneMail;

// MCP projection overrides (ADR 0016). Every /api operation becomes an MCP tool
// at /mcp unless it carries `x-mcp: { hidden: true }`; `x-mcp: { name }` overrides
// the default tool name (the operation id in snake_case, e.g. `contacts_list`).

// Segments are declared in the contract but have no handlers yet;
// a later ticket removes the `hidden` flag when it implements them.
@@extension(Segments.list, "x-mcp", #{ hidden: true });
@@extension(Segments.create, "x-mcp", #{ hidden: true });
@@extension(Segments.get, "x-mcp", #{ hidden: true });
@@extension(Segments.update, "x-mcp", #{ hidden: true });
@@extension(Segments.delete, "x-mcp", #{ hidden: true });

// API token management is never an agent capability.
@@extension(Auth.tokensCreate, "x-mcp", #{ hidden: true });
@@extension(Auth.tokensList, "x-mcp", #{ hidden: true });
@@extension(Auth.tokensDelete, "x-mcp", #{ hidden: true });
@@extension(Auth.tokensBootstrap, "x-mcp", #{ hidden: true });

// Names in 1mail vocabulary where the default would read poorly.
@@extension(Auth.meGet, "x-mcp", #{ name: "whoami" });
@@extension(Events.create, "x-mcp", #{ name: "events_record" });
@@extension(EventActions.list, "x-mcp", #{ name: "events_actions_list" });

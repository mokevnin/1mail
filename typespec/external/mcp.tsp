import "@typespec/openapi";

using TypeSpec.OpenAPI;

namespace OneMail;

// MCP projection overrides (ADR 0016). Every /api operation becomes an MCP tool
// at /mcp unless it carries `x-mcp: { hidden: true }`; `x-mcp: { name }` overrides
// the default tool name (the operation id in snake_case, e.g. `contacts_list`).

// API token management is never an agent capability.
@@extension(Auth.tokensCreate, "x-mcp", #{ hidden: true });
@@extension(Auth.tokensList, "x-mcp", #{ hidden: true });
@@extension(Auth.tokensDelete, "x-mcp", #{ hidden: true });
@@extension(Auth.tokensBootstrap, "x-mcp", #{ hidden: true });

// Names in 1mail vocabulary where the default would read poorly.
@@extension(Auth.meGet, "x-mcp", #{ name: "whoami" });
@@extension(Events.create, "x-mcp", #{ name: "events_record" });
@@extension(EventActions.list, "x-mcp", #{ name: "events_actions_list" });
@@extension(ContactsBatch.upsert, "x-mcp", #{ name: "contacts_upsert_batch" });
@@extension(EventsBatch.submit, "x-mcp", #{ name: "events_record_batch" });

// Read-only reference data (ADR 0016).
@@extension(CustomFields.list, "x-mcp", #{ name: "custom_fields_list" });
@@extension(SendingDomains.list, "x-mcp", #{ name: "sending_domains_list" });
@@extension(
  SendingDomainRates.list,
  "x-mcp",
  #{ name: "sending_domains_rates" }
);

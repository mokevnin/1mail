// Package mcpserver mounts the MCP surface at /mcp (ADR 0016). It is a projection
// of the external /api contract, not a second implementation: tools are generated
// from the embedded OpenAPI document and each call is dispatched in-process through
// the same ogen server with the caller's own Bearer token, so workspace scoping,
// token scopes, validation and RFC 7807 errors are the /api ones by construction.
package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	externalapi "github.com/mokevnin/1mail/gen/external"
	apiauth "github.com/mokevnin/1mail/internal/api/auth"
)

// apiPrefix is where the external ogen server is mounted.
const apiPrefix = "/api"

const instructions = "1mail marketing automation. Tools mirror the 1mail external API and act on " +
	"the workspace of the Bearer API token used to connect; results and errors are the API's."

// Authenticator validates a Bearer API token (the external API's security handler).
type Authenticator interface {
	HandleBearerAuth(ctx context.Context, op externalapi.OperationName, t externalapi.BearerAuth) (context.Context, error)
}

var _ Authenticator = (*apiauth.ExternalSecurityHandler)(nil)

// New builds the /mcp http.Handler: Streamable HTTP, stateless, one tool per
// non-hidden operation of the OpenAPI spec. api is the external API handler
// serving under /api; auth validates the connecting Bearer token.
func New(spec []byte, api http.Handler, auth Authenticator) (http.Handler, error) {
	ops, err := project(spec)
	if err != nil {
		return nil, fmt.Errorf("project MCP tools: %w", err)
	}
	srv := mcp.NewServer(&mcp.Implementation{Name: "1mail", Version: "1"}, &mcp.ServerOptions{Instructions: instructions})
	for _, op := range ops {
		srv.AddTool(op.tool, op.handler(api))
	}
	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)
	return requireToken(auth, streamable), nil
}

// requireToken rejects requests without a valid Bearer API token (401).
func requireToken(auth Authenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeProblem(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		if _, err := auth.HandleBearerAuth(r.Context(), "", externalapi.BearerAuth{Token: token}); err != nil {
			if errors.Is(err, apiauth.ErrUnauthorized) {
				writeProblem(w, http.StatusUnauthorized, "invalid bearer token")
				return
			}
			writeProblem(w, http.StatusInternalServerError, "internal server error")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeProblem(w http.ResponseWriter, code int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": code,
		"title":  http.StatusText(code),
		"detail": detail,
	})
}

// handler dispatches a tool call to the /api handler as the caller.
func (op *operation) handler(api http.Handler) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := map[string]any{}
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return toolError("arguments must be a JSON object: " + err.Error()), nil
			}
		}
		httpReq, err := op.request(ctx, args, req.Extra)
		if err != nil {
			return toolError(err.Error()), nil
		}
		rec := httptest.NewRecorder()
		api.ServeHTTP(rec, httpReq)
		return result(rec.Result()), nil
	}
}

// request builds the in-process /api request for the given tool arguments.
func (op *operation) request(ctx context.Context, args map[string]any, extra *mcp.RequestExtra) (*http.Request, error) {
	rest := make(map[string]any, len(args))
	for k, v := range args {
		rest[k] = v
	}

	path := op.path
	query := url.Values{}
	headers := http.Header{}
	for _, p := range op.params {
		v, ok := rest[p.name]
		if !ok || v == nil {
			continue
		}
		delete(rest, p.name)
		values := scalars(v)
		switch p.in {
		case "path":
			path = strings.ReplaceAll(path, "{"+p.name+"}", url.PathEscape(values[0]))
		case "query":
			query[p.name] = values
		case "header":
			headers.Set(p.name, values[0])
		}
	}
	if strings.Contains(path, "{") {
		return nil, errors.New("missing required path argument for " + op.path)
	}

	var body io.Reader
	if op.hasBody {
		payload := map[string]any{}
		switch {
		case op.bodyWhole:
			if v, ok := rest[bodyArgument]; ok {
				delete(rest, bodyArgument)
				raw, err := json.Marshal(v)
				if err != nil {
					return nil, err
				}
				body = bytes.NewReader(raw)
			}
		default:
			for field := range op.bodyFields {
				if v, ok := rest[field]; ok {
					payload[field] = v
					delete(rest, field)
				}
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				return nil, err
			}
			body = bytes.NewReader(raw)
		}
	}
	if len(rest) > 0 {
		unknown := make([]string, 0, len(rest))
		for k := range rest {
			unknown = append(unknown, k)
		}
		sort.Strings(unknown)
		return nil, errors.New("unknown arguments: " + strings.Join(unknown, ", "))
	}

	target := apiPrefix + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, op.method, target, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		httpReq.Header[k] = v
	}
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.Header.Set("Accept", "application/json")
	// The caller's own token: the /api security handler authenticates and scopes it.
	if extra != nil && extra.Header != nil {
		httpReq.Header.Set("Authorization", extra.Header.Get("Authorization"))
	}
	return httpReq, nil
}

// scalars renders an argument as one or more query/path/header strings.
func scalars(v any) []string {
	if list, ok := v.([]any); ok {
		out := make([]string, 0, len(list))
		for _, item := range list {
			out = append(out, scalar(item))
		}
		if len(out) == 0 {
			return []string{""}
		}
		return out
	}
	return []string{scalar(v)}
}

func scalar(v any) string {
	if f, ok := v.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

// result maps the /api response to a tool result: the body on success, the body
// (the RFC 7807 problem) as a tool error otherwise.
func result(resp *http.Response) *mcp.CallToolResult {
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	body := strings.TrimSpace(string(raw))
	if resp.StatusCode >= http.StatusBadRequest {
		return toolError(fmt.Sprintf("HTTP %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body))
	}
	if body == "" {
		body = fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: body}}}
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: msg}}}
}

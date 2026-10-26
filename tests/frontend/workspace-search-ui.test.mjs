import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { settleQuery } from "./helpers/query-router.mjs";
registerWebModuleHooks();
globalThis.vectorSearchNavigation = [];
globalThis.vectorSearchClosed = 0;
const stubs = {
  "@tanstack/react-router": `export const useParams=()=>({workspaceSlug:"acme",projectId:"project"});export const useRouter=()=>({navigate:async(value)=>{globalThis.vectorSearchNavigation.push(value)}});`,
  "@/hooks/use-power-k": `export const usePowerK=()=>({togglePowerKModal:()=>{globalThis.vectorSearchClosed++}});`,
  "@/components/issues/issue-detail/issue-identifier": `export const IssueIdentifier=()=>"APP-1";`,
  "./no-results-command": `export const PowerKModalNoSearchResultsCommand=()=>"No results";`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { Command } = await import(webRequire.resolve("cmdk").replace(/index\.js$/, "index.mjs"));
const { PowerKModalSearchMenu } = await import("../../apps/web/core/components/power-k/ui/modal/search-menu.tsx");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const snippet = "<img src=x onerror=bad> unique field needle";
const results = {
  results: {
    workspace: [],
    project: [],
    issue: [
      {
        id: "item",
        name: "Ordinary title",
        project_id: "project",
        project__identifier: "APP",
        sequence_id: 1,
        workspace__slug: "acme",
        type_id: "type",
        custom_property_match: { property_id: "field", property_title: "Notes", text: snippet },
      },
    ],
    page: [],
    module: [],
    cycle: [],
    issue_view: [],
  },
};
async function fixture(t, active = true) {
  const env = await createReactDomTestEnvironment();
  HTMLElement.prototype.scrollIntoView = () => {};
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  const memberKey = permissionKeys.workspaceMembership("alice", "acme");
  client.setQueryData(memberKey, { is_active: active, role: active ? 15 : null, updated_at: "v1" });
  client.setQueryData(permissionKeys.workspaceProjectRoles("alice", "acme", 15, "v1"), { project: 15 });
  t.after(async () => {
    await env.unmount();
    client.clear();
  });
  const render = (term = "unique field needle") =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(
          Command,
          { filter: (value, search) => (value.includes(search) ? 1 : 0) },
          React.createElement(Command.Input, { value: term, onValueChange: () => {} }),
          React.createElement(
            Command.List,
            null,
            React.createElement(PowerKModalSearchMenu, {
              activePage: null,
              context: {},
              isWorkspaceLevel: true,
              searchTerm: term,
              updateSearchTerm: () => {},
            })
          )
        )
      )
    );
  await render();
  await settleQuery();
  return { env, client, memberKey, render };
}
await test("server field-only results bypass title filtering, remain selectable and render snippets as text", async (t) => {
  t.mock.method(WorkspaceService.prototype, "searchWorkspace", async () => results);
  const f = await fixture(t);
  const item = f.env.container.querySelector("[cmdk-item]");
  assert.ok(item, "server match disappeared under the title filter");
  assert.equal(item.closest("[cmdk-group]").hidden, false);
  assert.match(item.textContent, /Ordinary title/);
  assert.ok(item.textContent.includes(snippet));
  assert.equal(item.querySelector("img"), null);
  await act(async () => item.click());
  assert.equal(globalThis.vectorSearchClosed, 1);
  assert.equal(globalThis.vectorSearchNavigation.length, 1);
  assert.equal(globalThis.vectorSearchNavigation[0].href, "/acme/browse/APP-1/");
});
await test("successful revoked membership renders access-unavailable, not endless loading or cached snippets", async (t) => {
  const search = t.mock.method(WorkspaceService.prototype, "searchWorkspace", async () => results);
  const f = await fixture(t, false);
  assert.match(f.env.container.querySelector('[role="alert"]').textContent, /current access/);
  assert.equal(f.env.container.querySelector(".animate-pulse"), null);
  assert.equal(f.env.container.querySelector("[cmdk-item]"), null);
  assert.equal(search.mock.calls.length, 0);
});
await test("changing the term or losing workspace membership immediately hides the previous snippet", async (t) => {
  t.mock.method(WorkspaceService.prototype, "searchWorkspace", async () => results);
  const f = await fixture(t);
  assert.ok(f.env.container.textContent.includes(snippet));
  await f.render("another term");
  assert.equal(f.env.container.textContent.includes(snippet), false);
  await act(async () => f.client.setQueryData(f.memberKey, { is_active: false, role: null, updated_at: "v2" }));
  await settleQuery();
  assert.match(f.env.container.querySelector('[role="alert"]').textContent, /current access/);
});

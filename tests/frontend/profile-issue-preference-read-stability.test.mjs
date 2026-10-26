import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";

const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { QueryClient, QueryClientProvider } = webRequire("@tanstack/react-query");
const requireFromWeb = createRequire(path.resolve("apps/web/package.json"));
const reactQueryModuleUrl = pathToFileURL(requireFromWeb.resolve("@tanstack/react-query")).href;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@tanstack/react-query") return { shortCircuit: true, url: reactQueryModuleUrl };
    return nextResolve(specifier, context);
  },
});
registerWebModuleHooks();
const { UserService } = await fromRoot("apps/web/core/services/user.service.ts");
const { identityKeys } = await fromRoot("apps/web/core/queries/identity.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { useIssueListQuery } = await import("../../apps/web/core/hooks/use-issue-list-query.ts");

await test("profile issue reads stay bounded across parent rerenders", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(async () => {
    await environment.unmount();
    client.clear();
  });
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  client.setQueryData(permissionKeys.workspaceMembership("alice", "acme"), {
    is_active: true,
    role: 15,
    updated_at: "membership-1",
  });
  client.setQueryData(permissionKeys.workspaceProjectRoles("alice", "acme", 15, "membership-1"), {});
  let readCount = 0;
  t.mock.method(UserService.prototype, "getUserProfileIssues", async () => {
    readCount++;
    return { results: [], total_count: 0, next_page_results: false, next_cursor: "", prev_cursor: "" };
  });
  const scope = {
    accountId: "alice",
    sessionGeneration: 0,
    workspaceSlug: "acme",
    workspaceRole: 15,
    membershipRevision: "membership-1",
    projectRoles: [],
  };
  const makeRead = () => ({
    scope: { ...scope },
    pagination: { canGroup: true, perPageCount: 30 },
    request: {
      family: "profile",
      input: {
        kind: "profile",
        workspaceSlug: "acme",
        userId: "alice",
        accountId: "alice",
        view: "created",
        richFilters: {},
        displayFilters: { layout: "list", order_by: "-created_at" },
        isCurrent: () => true,
        isReadScopeCurrent: () => true,
      },
      isReadCurrent: () => true,
    },
  });
  let read = makeRead();
  function Probe() {
    const query = useIssueListQuery(read);
    return React.createElement("output", null, query.isError ? "error" : query.isPending ? "pending" : "ready");
  }
  const tree = () => React.createElement(QueryClientProvider, { client }, React.createElement(Probe));
  await environment.render(tree());
  await act(async () => new Promise((resolve) => setTimeout(resolve, 0)));
  const rerenderWithEqualRead = async () => {
    read = makeRead();
    await act(async () => environment.render(tree()));
  };
  await rerenderWithEqualRead();
  await rerenderWithEqualRead();

  assert.equal(readCount, 1, "a structurally equal parent rerender reuses the accepted profile Query");
  assert.equal(environment.container.querySelector("output").textContent, "ready");
});

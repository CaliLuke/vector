import { createProjectQueryDomFixture } from "./helpers/query-dom.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
globalThis.typeDisplayReact = React;
const stubs = {
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@/hooks/use-current-user": `export const useCurrentUserId=()=>"alice";`,
  "@/hooks/use-permissions": `export const useProjectPermissions=()=>globalThis.typeDisplayFixture.permissions;`,
  "@/hooks/use-projects": `export const useWorkspaceProjectDetails=()=>({projects:[{id:"p",identifier:"PLAT"}]});`,
  "@/hooks/use-work-item-detail": `export const useWorkItemDetail=()=>({issue:globalThis.typeDisplayFixture.item});`,
  "@/components/issues/issue-detail/identifier-text": `export const IdentifierText=({identifier})=>globalThis.typeDisplayReact.createElement("span",null,identifier);`,
  "./controls": `export const TypeIcon=({type})=>globalThis.typeDisplayReact.createElement("svg",{"data-icon":type.icon,"data-color":type.color});`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const bootstrap = await createReactDomTestEnvironment();
const { FilterDisplayProperties } =
  await import("../../apps/web/core/components/issues/issue-layouts/filters/header/display-filters/display-properties.tsx");
const { IssueIdentifier } = await import("../../apps/web/core/components/issues/issue-detail/issue-identifier.tsx");
const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { workItemTypesService: service } = await import("../../apps/web/core/services/work-item-types.service.ts");
await bootstrap.unmount();
async function fixture(t) {
  const { env, client } = await createProjectQueryDomFixture(t, 15);
  const f = {
    env,
    client,
    calls: 0,
    item: { id: "item", type_id: "type", sequence_id: 42 },
    permissions: {
      isPending: false,
      isError: false,
      projectRole: 15,
      workspaceRole: 15,
      membershipQuery: { isSuccess: true, data: { updated_at: "v1" } },
    },
    catalog: {
      enabled: true,
      types: [{ id: "type", name: "Incident", icon: "incident", color: "#ff0000", active: false, imported: false }],
    },
  };
  globalThis.typeDisplayFixture = f;
  t.mock.method(service, "identities", async () => {
    f.calls++;
    return f.catalog;
  });
  f.render = (displayProperties, count = 1) =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(
          React.Fragment,
          null,
          ...Array.from({ length: count }, (_, i) =>
            React.createElement(IssueIdentifier, {
              key: i,
              workspaceSlug: "acme",
              projectId: "p",
              issueId: "item",
              displayProperties,
            })
          )
        )
      )
    );
  return f;
}
await test("type is independent of the key and retains paused or unimported identity", async (t) => {
  const f = await fixture(t);
  await f.render({ key: false, issue_type: true });
  assert.match(f.env.container.textContent, /Incident/);
  assert.doesNotMatch(f.env.container.textContent, /PLAT/);
  assert.equal(f.env.container.querySelector("svg").dataset.icon, "incident");
  await f.render({ key: true, issue_type: false });
  assert.match(f.env.container.textContent, /PLAT-42/);
  assert.doesNotMatch(f.env.container.textContent, /Incident/);
  await f.render({ key: true, issue_type: true });
  assert.match(f.env.container.textContent, /Incident/);
  assert.match(f.env.container.textContent, /PLAT-42/);
  await f.render({ key: false, issue_type: false });
  assert.equal(f.env.container.textContent, "");
});
await test("virtualized rows reuse the same fresh project catalog", async (t) => {
  const f = await fixture(t);
  await f.render({ key: false, issue_type: true }, 3);
  assert.equal(f.calls, 1);
  await f.render({ key: false, issue_type: true }, 10);
  assert.equal(f.calls, 1);
  assert.equal(f.env.container.querySelectorAll("svg").length, 10);
});
await test("disabled rollout hides the type badge without hiding the requested key", async (t) => {
  const f = await fixture(t);
  f.catalog.enabled = false;
  await f.render({ key: true, issue_type: true });
  assert.equal(f.env.container.textContent, "PLAT-42");
});
await test("untyped and unavailable saved identities are distinct", async (t) => {
  const f = await fixture(t);
  f.item.type_id = null;
  await f.render({ key: false, issue_type: true });
  assert.equal(f.env.container.textContent, "No type");
  f.item.type_id = "missing";
  await f.render({ key: false, issue_type: true });
  assert.equal(f.env.container.textContent, "Unavailable type");
});
await test("confirmed permission failure hides cached type names", async (t) => {
  const f = await fixture(t);
  await f.render({ key: false, issue_type: true });
  assert.match(f.env.container.textContent, /Incident/);
  f.permissions.isError = true;
  await f.render({ key: false, issue_type: true });
  assert.doesNotMatch(f.env.container.textContent, /Incident/);
});
await test("same account new session rejects an older catalog response", async (t) => {
  const f = await fixture(t),
    pending = Promise.withResolvers();
  let reads = 0;
  t.mock.method(service, "identities", async () =>
    ++reads === 1 ? pending.promise : { ...f.catalog, types: [{ ...f.catalog.types[0], name: "New session type" }] }
  );
  await f.render({ key: false, issue_type: true });
  await act(async () => advanceIdentitySession(f.client));
  await act(async () => pending.resolve(f.catalog));
  assert.match(f.env.container.textContent, /New session type/);
  assert.doesNotMatch(f.env.container.textContent, /Incident/);
});

function DisplayControlsAndIdentity() {
  const [properties, setProperties] = React.useState({ key: false, issue_type: false });
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(FilterDisplayProperties, {
      displayProperties: properties,
      displayPropertiesToRender: ["key", "issue_type"],
      handleUpdate: (change) => setProperties((previous) => ({ ...previous, ...change })),
    }),
    React.createElement(IssueIdentifier, {
      workspaceSlug: "acme",
      projectId: "p",
      issueId: "item",
      displayProperties: properties,
    })
  );
}

await test("the real Display control toggles type independently from ID", async (t) => {
  const f = await fixture(t);
  await f.env.render(
    React.createElement(QueryClientProvider, { client: f.client }, React.createElement(DisplayControlsAndIdentity))
  );
  const toggle = () =>
    [...f.env.container.querySelectorAll("button")].find(
      (button) => button.textContent === "issue.display.properties.issue_type"
    );
  assert.ok(toggle());
  assert.doesNotMatch(f.env.container.textContent, /Incident/);
  await act(async () => toggle().click());
  assert.match(f.env.container.textContent, /Incident/);
  assert.doesNotMatch(f.env.container.textContent, /PLAT-42/);
  await act(async () => toggle().click());
  assert.doesNotMatch(f.env.container.textContent, /Incident/);
});

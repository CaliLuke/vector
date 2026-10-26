import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";
import test from "node:test";

import { createModuleStub } from "./helpers/module-stubs.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

globalThis.analyticsModalFixture = { closes: 0, reads: 0, membershipError: true };
globalThis.analyticsModalReact = React;

const fixtureExpression = `globalThis.analyticsModalFixture`;
registerWebTestModules(
  {
    "@plane/propel/portal": `
        const React=globalThis.analyticsModalReact;
        export const EPortalWidth={FULL:"full",THREE_QUARTER:"three-quarter"};
        export const EPortalPosition={RIGHT:"right"};
        export const ModalPortal=({isOpen,children})=>isOpen?React.createElement("div",{"data-testid":"portal"},children):null;
      `,
    "@/hooks/use-current-user": `export const useCurrentUser=()=>({data:{id:"analytics-account"},isSuccess:true,isError:false,refetch:async()=>({})})`,
    "@/hooks/use-identity-generation": "export const useIdentityGeneration=()=>0",
    "@/hooks/use-permissions": `
        export const useWorkspacePermissions=()=>({
          workspaceRole:15,
          membershipQuery:{isSuccess:false,isError:${fixtureExpression}.membershipError,isPending:!${fixtureExpression}.membershipError,refetch:async()=>({})}
        });
        export const useWorkspaceProjectRoles=()=>({roles:undefined,rolesQuery:{isSuccess:false,isError:false,isPending:true,refetch:async()=>({})}});
      `,
  },
  (specifier, context, nextResolve) => {
    if (specifier === "./content" && context.parentURL?.includes("/work-items/modal/index.tsx"))
      return createModuleStub(`
        const React=globalThis.analyticsModalReact;
        export const WorkItemsModalMainContent=()=>{globalThis.analyticsModalFixture.reads++;return React.createElement("div",{"data-testid":"analytics-reads"},"Read panels");};
      `);
    return nextResolve(specifier, context);
  }
);

const { WorkItemsModal } = await import("../../apps/web/core/components/analytics/work-items/modal/index.tsx");
const { QueryClient, QueryClientProvider } = await import("@tanstack/react-query");

function renderModal(client, isOpen, onClose) {
  return React.createElement(
    QueryClientProvider,
    { client },
    React.createElement(WorkItemsModal, {
      workspaceSlug: "alpha",
      isOpen,
      onClose,
      projectDetails: { id: "project-a", name: "Alpha" },
    })
  );
}

void test("open analytics modal keeps its dismiss shell visible when permission loading fails", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = new QueryClient();
  const fixture = globalThis.analyticsModalFixture;
  fixture.closes = 0;
  fixture.reads = 0;
  fixture.membershipError = true;
  await environment.render(renderModal(client, true, () => fixture.closes++));
  assert.ok(environment.container.querySelector('[data-testid="portal"]'));
  assert.ok(environment.container.querySelector('[aria-label="Close analytics"]'));
  assert.match(environment.container.textContent, /Analytics permissions could not be loaded/);
  assert.equal(fixture.reads, 0);
  await React.act(async () => environment.container.querySelector('[aria-label="Close analytics"]').click());
  assert.equal(fixture.closes, 1);
  client.clear();
});

void test("closed analytics modal shows no permission UI and mounts no analytics reads", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = new QueryClient();
  const fixture = globalThis.analyticsModalFixture;
  fixture.reads = 0;
  fixture.membershipError = true;
  await environment.render(renderModal(client, false, () => fixture.closes++));
  assert.equal(environment.container.querySelector('[data-testid="portal"]'), null);
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
  assert.equal(fixture.reads, 0);
  client.clear();
});

import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, next) {
    if (context.parentURL?.endsWith("/issue-layout-HOC.tsx")) {
      if (specifier === "@/hooks/use-issue-list-type")
        return createModuleStub('export const useIssueListType=()=>"PROJECT";');
      if (specifier === "./empty-states")
        return createModuleStub('export const IssueLayoutEmptyState=()=>"Empty work items";');
    }
    return next(specifier, context);
  },
});
const { IssueLayoutHOC } = await import("../../apps/web/core/components/issues/issue-layouts/issue-layout-HOC.tsx");
await test("an authorized empty discovery group renders without suppressing errors or loading", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const props = {
    layout: "list",
    allowEmptyGroups: true,
    readState: { isPending: false, isError: false, issueCount: 0, retry: async () => {} },
  };
  const child = React.createElement("h2", null, "APP-1 Release — 0 items");
  await env.render(React.createElement(IssueLayoutHOC, props, child));
  assert.match(env.container.textContent, /APP-1 Release/);
  await env.render(React.createElement(IssueLayoutHOC, { ...props, allowEmptyGroups: false }, child));
  assert.equal(env.container.textContent, "Empty work items");
  await env.render(
    React.createElement(IssueLayoutHOC, { ...props, readState: { ...props.readState, isPending: true } }, child)
  );
  assert.equal(env.container.querySelector("h2"), null);
  assert.ok(env.container.querySelector('[aria-label="Loading work items"]'));
  await env.render(
    React.createElement(IssueLayoutHOC, { ...props, readState: { ...props.readState, isError: true } }, child)
  );
  assert.equal(env.container.querySelector("h2"), null);
  assert.ok(env.container.querySelector('[role="alert"]'));
});

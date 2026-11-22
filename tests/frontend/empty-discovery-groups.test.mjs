import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "./helpers/module-hooks.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
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

await test("an updating list keeps its rows mounted and temporarily blocks row interactions", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const row = React.createElement("button", { type: "button" }, "Work item");
  const readState = { isPending: false, isError: false, issueCount: 1, retry: async () => {} };
  const render = (isUpdatingResults) =>
    env.render(
      React.createElement(IssueLayoutHOC, { layout: "list", readState: { ...readState, isUpdatingResults } }, row)
    );

  await render(false);
  const original = env.container.querySelector("button");
  await render(true);
  assert.equal(env.container.querySelector("button"), original);
  assert.ok(original.closest("[inert]"));
  assert.ok(original.closest('[aria-busy="true"]')?.querySelector("output"));
  assert.equal(env.container.querySelector('[aria-label="Loading work items"]'), null);
  await render(false);
  assert.equal(env.container.querySelector("button"), original);
  assert.equal(original.closest("[inert]"), null);
});

await test("pagination and refresh alerts retry their own operation without remounting the results", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const row = React.createElement("article", null, "Work item");
  const calls = [];
  const readState = {
    isPending: false,
    isError: false,
    issueCount: 1,
    retry: async () => calls.push("blocking retry"),
    retryRefresh: async () => calls.push("refresh"),
    retryFailedPages: async () => calls.push("next page"),
  };
  const render = (read) =>
    env.render(React.createElement(IssueLayoutHOC, { layout: "list", readState: { ...readState, ...read } }, row));
  await render({});
  const original = env.container.querySelector("article");
  await render({ isPageError: true });
  const retryButton = () => env.container.querySelector('[role="alert"] button');
  await act(async () => retryButton().click());
  assert.equal(env.container.querySelector("article"), original);
  await render({ isRefreshError: true, isFetching: true });
  assert.equal(retryButton().disabled, true);
  await render({ isRefreshError: true });
  await act(async () => retryButton().click());
  assert.equal(env.container.querySelector("article"), original);
  assert.deepEqual(calls, ["next page", "refresh"]);
  await render({ isRefreshError: true, issueCount: 0 });
  assert.ok(retryButton(), "an empty accepted result still offers retry");
  assert.match(env.container.textContent, /Empty work items/);
});

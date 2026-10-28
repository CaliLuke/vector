import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { seedWorkspaceQuery } from "./helpers/workspace-query-fixture.mjs";
registerWebModuleHooks();
const state = { toasts: [] };
globalThis.descriptionAutosave = state;
globalThis.descriptionReact = React;
const stubs = {
  "@/components/editor/rich-text": `const React=globalThis.descriptionReact;export function RichTextEditor(props){globalThis.descriptionAutosave.editor=props;return React.createElement("output",null,props.value??"editing");}`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error"};export const setToast=toast=>globalThis.descriptionAutosave.toasts.push(toast);`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
async function fixture(t, save) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  await seedWorkspaceQuery(client);
  t.mock.timers.enable({ apis: ["setTimeout", "Date"] });
  t.after(async () => {
    await env.render(null);
    client.clear();
    await env.unmount();
    t.mock.timers.reset();
  });
  const { DescriptionInput } =
    await import("../../apps/web/core/components/editor/rich-text/description-input/root.tsx");
  const statuses = [];
  const props = {
    workspaceSlug: "workspace",
    projectId: "project",
    entityId: "first",
    fileAssetType: "issue_description",
    initialValue: "<p>original</p>",
    onSubmit: save,
    setIsSubmitting: (value) => statuses.push(value),
  };
  const render = (override) =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(DescriptionInput, { ...props, ...override })
      )
    );
  await render();
  return {
    env,
    statuses,
    render,
    advance: () =>
      act(async () => {
        t.mock.timers.tick(1600);
      }),
  };
}
const edit = (html) => act(async () => state.editor.onChange({ type: "doc" }, html, {}));
await test("description saves serialize edits, retain failed drafts, and retry without stale refetches", async (t) => {
  const calls = [],
    requests = [];
  const fixtureState = await fixture(t, async (value) => {
    calls.push(value);
    const request = Promise.withResolvers();
    requests.push(request);
    await request.promise;
  });
  const { env, statuses, render, advance } = fixtureState;
  await edit("<p>first</p>");
  await advance();
  assert.equal(calls.length, 1);
  await edit("<p>latest</p>");
  await advance();
  assert.equal(calls.length, 1, "second request must wait for the first");
  await render({ initialValue: "<p>stale refetch</p>" });
  assert.equal(state.editor.value, null);
  await act(async () => requests[0].resolve());
  await advance();
  assert.equal(calls.length, 2);
  assert.equal(calls[1].description_html, "<p>latest</p>");
  assert.equal(statuses.includes("submitted"), false);
  await act(async () => requests[1].reject(new Error("offline")));
  await advance();
  assert.equal(statuses.at(-1), "error");
  assert.ok(env.container.querySelector('[role="alert"]'));
  await act(async () => env.container.querySelector("button").click());
  assert.equal(calls[2].description_html, "<p>latest</p>");
  await act(async () => requests[2].resolve());
  await advance();
  assert.equal(statuses.at(-1), "submitted");
  assert.equal(state.editor.value, "<p>latest</p>");
  assert.equal(env.container.querySelector('[role="alert"]'), null);
});
await test("leaving an entity flushes its draft to the original callback once", async (t) => {
  const first = [],
    second = [];
  const { env, render, advance } = await fixture(t, async (value) => first.push(value));
  await edit("<p>unsaved first</p>");
  await render({ entityId: "second", initialValue: "<p>second</p>", onSubmit: async (value) => second.push(value) });
  await advance();
  assert.equal(first.length, 1);
  assert.equal(first[0].description_html, "<p>unsaved first</p>");
  assert.equal(second.length, 0);
  await edit("<p>unsaved second</p>");
  await env.render(null);
  await advance();
  assert.equal(second.length, 1);
  assert.equal(second[0].description_html, "<p>unsaved second</p>");
});

await test("leaving during a description retry does not submit its failed revision twice", async (t) => {
  const requests = [];
  const { env, advance } = await fixture(t, async () => {
    const request = Promise.withResolvers();
    requests.push(request);
    return request.promise;
  });
  await edit("<p>Retry this description</p>");
  await advance();
  await act(async () => requests[0].reject(new Error("offline")));
  await advance();
  await act(async () => env.container.querySelector("button").click());
  assert.equal(requests.length, 2);
  await env.render(null);
  await act(async () => requests[1].resolve());
  await advance();
  assert.equal(requests.length, 2);
});

await test("remote descriptions refresh pristine editors without replacing dirty or accepted local edits", async (t) => {
  const saves = [];
  const { render, advance } = await fixture(t, async (value) => saves.push(value));
  await render({ syncedDescription: "<p>remote update</p>" });
  assert.equal(state.editor.value, "<p>remote update</p>");
  await advance();
  assert.equal(saves.length, 0);

  await edit("<p>local draft</p>");
  await render({ syncedDescription: "<p>concurrent remote update</p>" });
  assert.equal(state.editor.value, null);
  await advance();
  assert.equal(saves.length, 1);
  assert.equal(saves[0].description_html, "<p>local draft</p>");
  assert.equal(state.editor.value, "<p>local draft</p>");

  await render({ syncedDescription: "<p>later remote update</p>" });
  assert.equal(state.editor.value, "<p>later remote update</p>");
  await advance();
  assert.equal(saves.length, 1);
});

await test("a queued autosave submits the newest draft once and consumes its pending debounce", async (t) => {
  const requests = [];
  const saved = [];
  const { advance } = await fixture(t, async ({ description_html }) => {
    saved.push(description_html);
    const request = Promise.withResolvers();
    requests.push(request);
    await request.promise;
  });
  await edit("<p>first</p>");
  await advance();
  await edit("<p>queued</p>");
  await advance();
  await edit("<p>latest</p>");
  await act(async () => requests[0].resolve());
  assert.deepEqual(saved, ["<p>first</p>", "<p>latest</p>"]);
  await act(async () => requests[1].resolve());
  await advance();
  assert.equal(saved.length, 2);
});

import { registerHooks } from "node:module";
import test from "node:test";
import assert from "node:assert/strict";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedWorkspaceQuery } from "./helpers/workspace-query-fixture.mjs";

registerWebModuleHooks();
const state = { toasts: [] };
globalThis.titleAutosave = state;
globalThis.titleReact = React;
registerHooks({
  resolve: (specifier, context, next) =>
    resolvePackageStub(
      specifier,
      {
        "@/components/editor/rich-text": `export function RichTextEditor(props){globalThis.titleAutosave.description=props;globalThis.titleReact.useLayoutEffect(()=>{if(globalThis.titleAutosave.startupBody)props.onChange({},globalThis.titleAutosave.startupBody,{});},[]);return null;}`,
        "@plane/ui": `export function TextArea(props){return globalThis.titleReact.createElement("textarea",props)}`,
        "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
        "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error"};export const setToast=toast=>globalThis.titleAutosave.toasts.push(toast);`,
      },
      context,
      next
    ),
});

async function titleFixture(t, updateText, saveDescription) {
  const env = await createReactDomTestEnvironment();
  const { IssueTitleInput } = await import("../../apps/web/core/components/issues/title-input.tsx");
  const { IssueDescriptionInput } = await import("../../apps/web/core/components/issues/description-input.tsx");
  const { useItemEditLease } = await import("../../apps/web/core/components/issues/use-item-edit-coordination.ts");
  let lease;
  function Lease({ item }) {
    const current = useItemEditLease(`workspace/project/${item}`);
    React.useLayoutEffect(() => {
      lease = current;
    });
    return null;
  }
  const client = new QueryClient();
  await seedWorkspaceQuery(client);
  const statuses = [];
  const props = {
    workspaceSlug: "workspace",
    projectId: "project",
    issueId: "first",
    value: "Original",
    issueOperations: { updateText },
    setIsSubmitting: (status) => statuses.push(status),
  };
  t.mock.timers.enable({ apis: ["setTimeout", "Date"] });
  t.after(async () => {
    await env.unmount();
    client.clear();
    t.mock.timers.reset();
  });
  const render = (overrides, visible = true, extraTitle = false) =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        visible && React.createElement(IssueTitleInput, { ...props, ...overrides }),
        extraTitle && React.createElement(IssueTitleInput, { ...props, ...overrides, className: "second-title" }),
        React.createElement(Lease, { item: overrides?.issueId ?? props.issueId }),
        saveDescription &&
          React.createElement(IssueDescriptionInput, {
            workspaceSlug: "workspace",
            projectId: "project",
            entityId: "first",
            fileAssetType: "issue_description",
            initialValue: "<p>Original</p>",
            issueOperations: { updateText: (_workspace, _project, _issue, value) => saveDescription(value) },
            setIsSubmitting: () => {},
          })
      )
    );
  await render();
  const advance = (ms = 1500) => act(async () => t.mock.timers.tick(ms));
  const input = () => env.container.querySelector("textarea");
  const edit = (value) =>
    act(async () => {
      Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, "value").set.call(input(), value);
      input().dispatchEvent(new window.Event("input", { bubbles: true }));
    });
  const blur = () => act(async () => input().dispatchEvent(new window.FocusEvent("focusout", { bubbles: true })));
  return { env, client, statuses, render, advance, edit, blur, input, lease: () => lease };
}

await test("title debounce serializes revisions and retries the retained draft after failure", async (t) => {
  const writes = [];
  const requests = [];
  const f = await titleFixture(t, async (...args) => {
    writes.push(args);
    const request = Promise.withResolvers();
    requests.push(request);
    await request.promise;
  });
  await f.edit("First edit");
  await f.advance(1499);
  assert.equal(writes.length, 0);
  await f.advance(1);
  assert.equal(writes.length, 1);
  await f.edit("Latest edit");
  await f.advance();
  assert.equal(writes.length, 1);
  await f.render({ value: "Stale refetch" });
  assert.equal(f.input().value, "Latest edit");
  await act(async () => requests[0].resolve());
  await f.advance(0);
  assert.equal(writes.length, 2);
  assert.equal(writes[1][3].name, "Latest edit");
  assert.equal(f.statuses.includes("submitted"), false);
  await act(async () => requests[1].reject(new Error("offline")));
  await f.advance(0);
  assert.equal(f.statuses.at(-1), "error");
  assert.equal(f.input().value, "Latest edit");
  assert.ok(f.env.container.querySelector('[role="alert"]'));
  await act(async () => f.env.container.querySelector("button").click());
  assert.equal(writes[2][3].name, "Latest edit");
  await act(async () => requests[2].resolve());
  await f.advance(0);
  assert.equal(f.statuses.at(-1), "submitted");
  assert.equal(f.env.container.querySelector('[role="alert"]'), null);
});

await test("blank title restores the saved value on blur; whitespace is trimmed before writing", async (t) => {
  const writes = [];
  const f = await titleFixture(t, async (...args) => writes.push(args));
  await f.edit("   ");
  await f.advance();
  assert.equal(writes.length, 0);
  await f.blur();
  assert.equal(f.input().value, "Original");
  assert.equal(f.statuses.at(-1), "saved");
  await f.edit("  Renamed  ");
  await f.blur();
  assert.equal(f.input().value, "Renamed");
  await f.advance();
  assert.equal(writes.length, 1);
  assert.equal(writes[0][3].name, "Renamed");
});

await test("switching issue flushes only its unsent title and never reports late success to the replacement", async (t) => {
  const writes = [];
  const requests = [];
  const f = await titleFixture(t, async (...args) => {
    writes.push(args);
    const request = Promise.withResolvers();
    requests.push(request);
    await request.promise;
  });
  await f.edit("Leaving first");
  await f.render({ issueId: "second", value: "Second" });
  assert.equal(writes.length, 1);
  assert.equal(writes[0][2], "first");
  assert.equal(writes[0][3].name, "Leaving first");
  assert.equal(f.input().value, "Second");
  const statusCount = f.statuses.length;
  await act(async () => requests[0].resolve());
  await f.advance();
  assert.equal(f.statuses.length, statusCount);
  await f.edit("Leaving second");
  await f.advance();
  assert.equal(writes.length, 2);
  await f.env.render(null);
  await act(async () => requests[1].resolve());
  await f.advance();
  assert.equal(writes.length, 2, "unmount must not duplicate an already queued title");
});

await test("an old failed title request cannot mark a newer draft failed or replace it", async (t) => {
  const requests = [];
  const f = await titleFixture(t, async () => {
    const request = Promise.withResolvers();
    requests.push(request);
    await request.promise;
  });
  await f.edit("First");
  await f.advance();
  await f.edit("Second");
  await act(async () => requests[0].reject(new Error("old failure")));
  await f.advance(0);
  assert.equal(f.input().value, "Second");
  assert.equal(f.statuses.at(-1), "submitting");
  assert.equal(f.env.container.querySelector('[role="alert"]'), null);
  await f.advance();
  await act(async () => requests[1].resolve());
  await f.advance(0);
  assert.equal(f.statuses.at(-1), "submitted");
});

await test("title and description status keeps the unload guard until both writes succeed", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const { useTextSaveStatus } = await import("../../apps/web/core/components/issues/use-text-save-status.ts");
  let statuses;
  function Status({ scope }) {
    const current = useTextSaveStatus(scope);
    React.useEffect(() => {
      statuses = current;
    });
    return React.createElement("output", null, current.status);
  }
  await env.render(React.createElement(Status, { scope: "first" }));

  await act(async () => {
    statuses.setTitleStatus("submitting");
    statuses.setDescriptionStatus("submitting");
  });
  await act(async () => statuses.setDescriptionStatus("submitted"));
  assert.equal(env.container.textContent, "submitting");
  assert.equal(guarded(), true);
  await act(async () => statuses.setTitleStatus("error"));
  await act(async () => t.mock.timers.tick(3000));
  assert.equal(env.container.textContent, "error");
  assert.equal(guarded(), true);
  await act(async () => statuses.setTitleStatus("submitted"));
  assert.equal(guarded(), false);
  await act(async () => t.mock.timers.tick(2000));
  assert.equal(env.container.textContent, "saved");
  await act(async () => statuses.setTitleStatus("error"));
  await env.render(React.createElement(Status, { scope: "second" }));
  assert.equal(env.container.textContent, "saved");
  assert.equal(guarded(), false);
});

function guarded() {
  const event = new window.Event("beforeunload", { cancelable: true });
  window.dispatchEvent(event);
  return event.defaultPrevented;
}

await test("title and description writes share a Query scope so one rollback cannot race the other", async (t) => {
  const titleWrite = Promise.withResolvers();
  const saved = { title: "Original", description: "<p>Original</p>" };
  let descriptionStarted = false;
  const f = await titleFixture(
    t,
    async () => {
      saved.title = "Optimistic title";
      try {
        await titleWrite.promise;
      } catch (error) {
        saved.title = "Original";
        throw error;
      }
    },
    async (value) => {
      descriptionStarted = true;
      saved.description = value.description_html;
    }
  );
  await f.edit("Optimistic title");
  await f.advance();
  await act(async () => state.description.onChange({}, "<p>Edited description</p>", {}));
  await f.advance();
  assert.equal(descriptionStarted, false);
  await act(async () => titleWrite.reject(new Error("title rejected")));
  await f.advance(0);
  assert.equal(descriptionStarted, true);
  assert.deepEqual(saved, { title: "Original", description: "<p>Edited description</p>" });
  assert.equal(f.statuses.at(-1), "error");
});

await test("leaving during a title retry does not enqueue the failed revision a second time", async (t) => {
  const requests = [];
  const f = await titleFixture(t, async () => {
    const request = Promise.withResolvers();
    requests.push(request);
    return request.promise;
  });
  await f.edit("Retry this title");
  await f.advance();
  await act(async () => requests[0].reject(new Error("offline")));
  await f.advance(0);
  await act(async () => f.env.container.querySelector("button").click());
  assert.equal(requests.length, 2);
  await f.env.render(null);
  await act(async () => requests[1].resolve());
  await f.advance();
  assert.equal(requests.length, 2);
});

await test("trimming on blur does not duplicate a pending write of the same normalized title", async (t) => {
  const request = Promise.withResolvers();
  const writes = [];
  const f = await titleFixture(t, async (...args) => {
    writes.push(args);
    await request.promise;
  });
  await f.edit("  Trim this  ");
  await f.advance();
  await f.blur();
  assert.equal(f.input().value, "Trim this");
  await f.advance();
  await act(async () => request.resolve());
  await f.advance();
  assert.equal(writes.length, 1);
  assert.equal(writes[0][3].name, "Trim this");
  assert.equal(f.statuses.at(-1), "submitted");
});

await test("a modal lease waits for debounce and failed saves, then protects title and description", async (t) => {
  const response = Promise.withResolvers();
  let fail = true;
  const descriptions = [];
  const f = await titleFixture(
    t,
    async () => {
      await response.promise;
      if (fail) throw new Error("offline");
    },
    async (value) => descriptions.push(value)
  );
  await f.edit("Local draft");
  assert.equal(f.lease().acquire(), false);
  await f.advance();
  assert.equal(f.lease().acquire(), false);
  await act(async () => response.resolve());
  await f.advance(0);
  assert.equal(f.lease().acquire(), false);
  fail = false;
  await act(async () => f.env.container.querySelector("button").click());
  await f.advance(0);
  const staleDescription = state.description;
  await act(async () => assert.equal(f.lease().acquire(), true));
  assert.equal(f.input(), null);
  assert.equal(state.description.editable, false);
  await act(async () => staleDescription.onChange({}, "<p>Imperative restore</p>", {}));
  await f.advance();
  assert.equal(descriptions.length, 0);
  await act(async () => f.lease().release());
  assert.equal(f.input().value, "Local draft");
});
await test("one surface finishing cannot clear a second surface's unsent title", async (t) => {
  const saved = Promise.withResolvers();
  const f = await titleFixture(t, async () => saved.promise);
  await f.render({}, true, true);
  await f.edit("First surface");
  await f.advance();
  await act(async () => {
    const second = f.env.container.querySelector(".second-title");
    Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, "value").set.call(second, "Second surface");
    second.dispatchEvent(new window.Event("input", { bubbles: true }));
    saved.resolve();
  });
  await f.advance(0);
  assert.equal(f.lease().acquire(), false);
  await f.advance();
  await act(async () => assert.equal(f.lease().acquire(), true));
});
await test("an unmounted editor's flush stays visible to lease acquisition", async (t) => {
  const saved = Promise.withResolvers();
  const f = await titleFixture(t, async () => saved.promise);
  await f.edit("Flush on unmount");
  await f.render({}, false);
  assert.equal(f.lease().acquire(), false);
  await act(async () => saved.resolve());
  await f.advance(0);
  await act(async () => assert.equal(f.lease().acquire(), true));
});
await test("a session change drops old unsent title and description without sending them", async (t) => {
  const writes = [],
    descriptions = [];
  const f = await titleFixture(
    t,
    async (...args) => writes.push(args),
    async (value) => descriptions.push(value)
  );
  const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
  await f.edit("Old session title");
  await act(async () => state.description.onChange({}, "<p>Old session description</p>", {}));
  assert.equal(f.lease().acquire(), false);
  await act(async () => advanceIdentitySession(f.client));
  await f.advance();
  assert.equal(writes.length, 0);
  assert.equal(descriptions.length, 0);
  assert.equal(f.input().value, "Original");
  assert.equal(f.statuses.at(-1), "saved");
  await act(async () => assert.equal(f.lease().acquire(), true));
});
await test("old lease cleanup cannot unlock a replacement item", async (t) => {
  const f = await titleFixture(t, async () => {});
  const first = f.lease();
  await act(async () => assert.equal(first.acquire(), true));
  await f.render({ issueId: "second", value: "Second" });
  await act(async () => assert.equal(f.lease().acquire(), true));
  await act(async () => first.release());
  assert.equal(f.input(), null);
  assert.equal(f.lease().held, true);
});

await test("a child's initial layout edit is visible before passive owner registration", async (t) => {
  state.startupBody = "<p>Initial migrated description</p>";
  t.after(() => {
    state.startupBody = undefined;
  });
  const writes = [];
  const f = await titleFixture(
    t,
    async () => {},
    async (value) => writes.push(value)
  );
  assert.equal(f.lease().acquire(), false);
  await f.advance();
  assert.equal(writes.length, 1);
  await act(async () => assert.equal(f.lease().acquire(), true));
});

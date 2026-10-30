import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { registerIssueRealEditor } from "./helpers/issue-real-editor.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
registerIssueRealEditor();
const state = { React, toasts: [] };
globalThis.editorClipboardFixture = state;
const react = "const React=globalThis.editorClipboardFixture.React;";
const stringSource = new URL("../../packages/utils/src/string.ts", import.meta.url).href;
const stubs = {
  "@/hooks/use-work-item-detail":
    'export const useWorkItemDetail=()=>({issue:{id:"issue",project_id:"project",sequence_id:3}});',
  "@/hooks/use-projects":
    'export const useWorkspaceProjectDetails=()=>({projects:[{id:"project",identifier:"DEMO"}]});',
  "@plane/utils/work-item":
    "export const generateWorkItemLink=({workspaceSlug,projectId,issueId})=>`/${workspaceSlug}/projects/${projectId}/issues/${issueId}`;",

  "@tiptap/react":
    react +
    'export const NodeViewWrapper=({children})=>React.createElement("div",null,children);export const NodeViewContent=()=>null;',
  lowlight: "export const common={};export const createLowlight=()=>({register(){}});",
  "highlight.js/lib/languages/typescript": "export default {};",
  "@plane/propel/tooltip": "export const Tooltip=({children})=>children;",
  "@plane/propel/icons":
    react +
    'export const CopyIcon=()=>React.createElement("span",null,"Copy");export const GlobeIcon=()=>null;export const EditIcon=()=>null;',
  "lucide-react":
    react + 'export const CheckIcon=()=>React.createElement("span",null,"Copied");export const Link2Off=()=>null;',
  "@plane/propel/toast":
    'export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=t=>globalThis.editorClipboardFixture.toasts.push(t);',
};
registerHooks({
  resolve(specifier, context, next) {
    if (specifier === "@plane/utils/string") return next(stringSource, context);
    return resolvePackageStub(specifier, stubs, context, next);
  },
});
const { CodeBlockComponent } = await import("../../packages/editor/src/core/extensions/code/code-block-node-view.tsx");
const { CreateIssueToastActionItems } =
  await import("../../apps/web/core/components/issues/create-issue-toast-action-items.tsx");
const { LinkPreview } = await import("../../packages/editor/src/core/components/links/link-preview.tsx");
async function setup(t) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  state.toasts.length = 0;
  let pending = Promise.withResolvers();
  const writes = [];
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: {
      writeText: (text) => {
        writes.push(text);
        return pending.promise;
      },
    },
  });
  return {
    env,
    writes,
    get pending() {
      return pending;
    },
    retry() {
      pending = Promise.withResolvers();
    },
  };
}
function code(value = "original") {
  return React.createElement(CodeBlockComponent, { node: { attrs: { id: "block" }, textContent: value } });
}
void test("code copying suppresses its click synchronously, reports denial and permits retry", async (t) => {
  const f = await setup(t);
  let bubbles = 0;
  await f.env.render(React.createElement("div", { onClick: () => bubbles++ }, code()));
  const event = new MouseEvent("click", { bubbles: true, cancelable: true });
  await act(async () => f.env.container.querySelector("button").dispatchEvent(event));
  assert.equal(event.defaultPrevented, true);
  assert.equal(bubbles, 0);
  assert.deepEqual(f.writes, ["original"]);
  assert.equal(f.env.container.textContent, "Copy");
  await act(async () => f.pending.reject(new Error("clipboard denied")));
  assert.equal(state.toasts.at(-1).type, "error");
  assert.equal(f.env.container.textContent, "Copy");
  f.retry();
  await act(async () => f.env.container.querySelector("button").click());
  await act(async () => f.pending.resolve());
  assert.equal(f.env.container.textContent, "Copied");
});
for (const outcome of ["resolve", "reject"])
  void test(`code copy ignores ${outcome} after content changes`, async (t) => {
    const f = await setup(t);
    await f.env.render(code());
    await act(async () => f.env.container.querySelector("button").click());
    await f.env.render(code("replacement"));
    await act(async () => f.pending[outcome](new Error("denied")));
    assert.equal(f.env.container.textContent, "Copy");
    assert.deepEqual(state.toasts, []);
  });
void test("link preview stays open on denial and closes only after successful retry", async (t) => {
  const f = await setup(t);
  let closes = 0;
  const editor = { isEditable: false };
  const props = { editor, from: 1, to: 4, url: "https://example.com/", closeLinkView: () => closes++ };
  await f.env.render(React.createElement(LinkPreview, { viewProps: props, switchView() {} }));
  await act(async () => f.env.container.querySelector("button").click());
  assert.equal(closes, 0);
  await act(async () => f.pending.reject(new Error("denied")));
  assert.equal(closes, 0);
  assert.equal(state.toasts.at(-1).type, "error");
  f.retry();
  await act(async () => f.env.container.querySelector("button").click());
  await act(async () => f.pending.resolve());
  assert.equal(closes, 1);
  assert.deepEqual(f.writes, [props.url, props.url]);
});
for (const outcome of ["resolve", "reject"])
  void test(`link preview ignores ${outcome} after switching links`, async (t) => {
    const f = await setup(t);
    let closes = 0;
    const props = {
      editor: { isEditable: false },
      from: 1,
      to: 4,
      url: "https://example.com/old",
      closeLinkView: () => closes++,
    };
    await f.env.render(React.createElement(LinkPreview, { viewProps: props, switchView() {} }));
    await act(async () => f.env.container.querySelector("button").click());
    await f.env.render(
      React.createElement(LinkPreview, { viewProps: { ...props, url: "https://example.com/new" }, switchView() {} })
    );
    await act(async () => f.pending[outcome](new Error("denied")));
    assert.equal(closes, 0);
    assert.deepEqual(state.toasts, []);
  });
void test("a second accepted code copy resets the copied indicator deadline", async (t) => {
  const f = await setup(t);
  t.mock.timers.enable({ apis: ["setTimeout"] });
  await f.env.render(code());
  await act(async () => f.env.container.querySelector("button").click());
  await act(async () => f.pending.resolve());
  await act(async () => t.mock.timers.tick(500));
  f.retry();
  await act(async () => f.env.container.querySelector("button").click());
  await act(async () => f.pending.resolve());
  await act(async () => t.mock.timers.tick(500));
  assert.equal(f.env.container.textContent, "Copied");
  await act(async () => t.mock.timers.tick(500));
  assert.equal(f.env.container.textContent, "Copy");
});
for (const outcome of ["resolve", "reject"])
  void test(`code copy ignores ${outcome} after unmount`, async (t) => {
    const f = await setup(t);
    await f.env.render(code());
    await act(async () => f.env.container.querySelector("button").click());
    await f.env.render(null);
    await act(async () => f.pending[outcome](new Error("denied")));
    assert.deepEqual(state.toasts, []);
  });

function createdIssue(issueId = "issue") {
  return React.createElement(CreateIssueToastActionItems, { workspaceSlug: "acme", projectId: "project", issueId });
}
void test("created work-item copy suppresses its click before awaiting and reports denial", async (t) => {
  const f = await setup(t);
  let bubbles = 0;
  await f.env.render(React.createElement("div", { onClick: () => bubbles++ }, createdIssue()));
  const event = new MouseEvent("click", { bubbles: true, cancelable: true });
  await act(async () => f.env.container.querySelector("button").dispatchEvent(event));
  assert.equal(event.defaultPrevented, true);
  assert.equal(bubbles, 0);
  assert.deepEqual(f.writes, ["http://localhost/acme/projects/project/issues/issue"]);
  assert.equal(f.env.container.textContent.includes("Copied!"), false);
  await act(async () => f.pending.reject(new Error("denied")));
  assert.equal(state.toasts.at(-1).type, "error");
  f.retry();
  await act(async () => f.env.container.querySelector("button").click());
  await act(async () => f.pending.resolve());
  assert.equal(f.env.container.textContent.includes("Copied!"), true);
});
for (const outcome of ["resolve", "reject"])
  void test(`created work-item copy ignores ${outcome} after changing work items`, async (t) => {
    const f = await setup(t);
    await f.env.render(createdIssue());
    await act(async () => f.env.container.querySelector("button").click());
    await f.env.render(createdIssue("replacement"));
    await act(async () => f.pending[outcome](new Error("denied")));
    assert.equal(f.env.container.textContent.includes("Copied!"), false);
    assert.deepEqual(state.toasts, []);
  });

for (const outcome of ["resolve", "reject"]) {
  void test(`a newer code copy owns feedback when an older request later ${outcome}s`, async (t) => {
    const f = await setup(t);
    await f.env.render(code());
    const first = f.pending;
    await act(async () => f.env.container.querySelector("button").click());
    f.retry();
    await act(async () => f.env.container.querySelector("button").click());
    await act(async () => f.pending.resolve());
    assert.equal(f.env.container.textContent, "Copied");
    await act(async () => first[outcome](new Error("older copy denied")));
    assert.equal(f.env.container.textContent, "Copied");
    assert.deepEqual(state.toasts, []);
  });
}

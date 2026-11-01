import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const state = { requests: [], toasts: [], response: undefined };
globalThis.editorAIResponse = state;
const stubs = {
  "@/services/ai.service":
    "export class AIService { performEditorTask(workspace, payload) { const request = Promise.withResolvers(); globalThis.editorAIResponse.requests.push({workspace, payload, ...request}); return request.promise; } }",
  "@plane/propel/toast":
    "export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = toast => globalThis.editorAIResponse.toasts.push(toast);",
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { useEditorAIResponse } =
  await import("../../apps/web/core/components/pages/editor/ai/use-editor-ai-response.ts");
function Harness({ scope = "session-1" }) {
  const response = useEditorAIResponse("workspace", scope);
  React.useLayoutEffect(() => {
    state.response = response;
  }, [response]);
  return React.createElement(
    "output",
    null,
    JSON.stringify({ response: response.response, pending: response.isRegenerating })
  );
}
async function fixture(t) {
  state.requests = [];
  state.toasts = [];
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(Harness));
  return env;
}
const payload = { task: "rephrase", text_input: "selected text" };
async function request(clear = true) {
  const result = { completion: Promise.resolve(false) };
  await act(async () => {
    result.completion = state.response.generateResponse(payload, clear);
  });
  return result;
}
void test("AI rejection clears pending, reports one failure, and permits retry", async (t) => {
  const env = await fixture(t);
  const first = await request();
  assert.equal(state.response.isRegenerating, true);
  await act(async () => {
    state.requests[0].reject(new Error("private transport detail"));
    await first.completion;
  });
  assert.equal(state.response.isRegenerating, false);
  assert.equal(state.toasts.length, 1);
  assert.doesNotMatch(JSON.stringify(state.toasts), /private transport/);
  const second = await request();
  await act(async () => {
    state.requests[1].resolve({ response: "revised" });
    assert.equal(await second.completion, true);
  });
  assert.match(env.container.textContent, /revised/);
  assert.equal(state.response.isRegenerating, false);
  assert.deepEqual(state.requests[1].payload, payload);
});
void test("only the latest AI request may publish response or clear pending", async (t) => {
  await fixture(t);
  const first = await request();
  const second = await request();
  await act(async () => {
    state.requests[0].resolve({ response: "old" });
    assert.equal(await first.completion, false);
  });
  assert.equal(state.response.isRegenerating, true);
  assert.equal(state.response.response, undefined);
  await act(async () => {
    state.requests[1].resolve({ response: "latest" });
    await second.completion;
  });
  assert.equal(state.response.response, "latest");
  assert.equal(state.response.isRegenerating, false);
});
for (const [scope, fail] of [
  ["closed", false],
  ["new-session", true],
  [null, true],
])
  void test(`AI ignores late completion after ${scope ?? "unmount"}`, async (t) => {
    const env = await fixture(t);
    const current = await request();
    const pending = state.requests.at(-1);
    await env.render(scope ? React.createElement(Harness, { key: scope, scope }) : null);
    await act(async () => {
      if (fail) pending.reject(new Error("late"));
      else pending.resolve({ response: "late" });
      assert.equal(await current.completion, false);
    });
    assert.doesNotMatch(env.container.textContent, /late/);
    assert.deepEqual(state.toasts, []);
  });

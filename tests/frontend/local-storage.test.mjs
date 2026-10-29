import assert from "node:assert/strict";
import test from "node:test";
import { React, act, webRequire } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const { renderToString } = webRequire("react-dom/server");
registerWebModuleHooks();
const { useLocalStorage } = await import("../../packages/hooks/src/use-local-storage.tsx");

function Preference({ name = "first", storageKey = "panel", fallback = "closed", observed }) {
  const preference = useLocalStorage(storageKey, fallback);
  React.useImperativeHandle(observed[name], () => preference);
  return React.createElement("output", { "data-name": name }, JSON.stringify(preference.storedValue));
}

function createPreferenceRefs() {
  return { first: React.createRef(), second: React.createRef(), unrelated: React.createRef() };
}

async function setup(t) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  return env;
}

async function renderLinkedPreferences(env) {
  const observed = createPreferenceRefs();
  await env.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(Preference, { observed }),
      React.createElement(Preference, { name: "second", observed }),
      React.createElement(Preference, { name: "unrelated", storageKey: "another-panel", observed })
    )
  );
  return observed;
}

function readOnlyStorage(values) {
  return {
    getItem: (key) => values[key] ?? null,
    setItem() {
      throw new window.DOMException("Storage full", "QuotaExceededError");
    },
    removeItem() {
      throw new window.DOMException("Storage denied", "SecurityError");
    },
  };
}

void test("server rendering uses the preference default without browser storage", () => {
  const observed = createPreferenceRefs();
  const markup = renderToString(React.createElement(Preference, { observed }));
  assert.match(markup, /closed/);
});

void test("saved false values survive initialization instead of becoming the default", async (t) => {
  const env = await setup(t);
  window.localStorage.setItem("panel", "false");
  const observed = createPreferenceRefs();
  await env.render(React.createElement(Preference, { fallback: true, observed }));
  assert.equal(observed.first.current.storedValue, false);
});

void test("malformed saved JSON is removed and the preference falls back", async (t) => {
  const env = await setup(t);
  window.localStorage.setItem("panel", "{");
  const observed = createPreferenceRefs();
  await env.render(React.createElement(Preference, { observed }));
  assert.equal(observed.first.current.storedValue, "closed");
  assert.equal(window.localStorage.getItem("panel"), null);
});

void test("same-page preferences share writes and clearing removes the saved value", async (t) => {
  const env = await setup(t);
  const observed = await renderLinkedPreferences(env);
  await act(async () => observed.first.current.setValue("open"));
  assert.equal(window.localStorage.getItem("panel"), '"open"');
  assert.equal(observed.first.current.storedValue, "open");
  assert.equal(observed.second.current.storedValue, "open");
  assert.equal(observed.unrelated.current.storedValue, "closed");
  await act(async () => observed.first.current.clearValue());
  assert.equal(window.localStorage.getItem("panel"), null);
  assert.equal(observed.first.current.storedValue, null);
  assert.equal(observed.second.current.storedValue, "closed");
});

for (const [name, descriptor, initial] of [
  [
    "denied storage retains transient choices and synchronizes same-page preferences",
    {
      get() {
        throw new window.DOMException("Storage denied", "SecurityError");
      },
    },
    "closed",
  ],
  [
    "failed writes and corrupt-value cleanup retain transient choices",
    { value: readOnlyStorage({ panel: "{" }) },
    "closed",
  ],
  [
    "failed writes and removals do not restore the previously saved preference",
    { value: readOnlyStorage({ panel: '"saved"' }) },
    "saved",
  ],
]) {
  void test(name, async (t) => {
    const env = await setup(t);
    Object.defineProperty(window, "localStorage", { configurable: true, ...descriptor });
    const observed = await renderLinkedPreferences(env);
    assert.equal(observed.first.current.storedValue, initial);
    await act(async () => observed.first.current.setValue("open"));
    assert.equal(observed.first.current.storedValue, "open");
    assert.equal(observed.second.current.storedValue, "open");
    assert.equal(observed.unrelated.current.storedValue, "closed");
    await act(async () => observed.first.current.clearValue());
    assert.equal(observed.first.current.storedValue, null);
    assert.equal(observed.second.current.storedValue, "closed");
  });
}

void test("changing preference keys reads the new scope and detaches the previous key", async (t) => {
  const env = await setup(t);
  window.localStorage.setItem("workspace-a", '"open"');
  window.localStorage.setItem("workspace-b", '"closed"');
  const observed = createPreferenceRefs();
  await env.render(React.createElement(Preference, { storageKey: "workspace-a", observed }));
  assert.equal(observed.first.current.storedValue, "open");
  await env.render(React.createElement(Preference, { storageKey: "workspace-b", observed }));
  assert.equal(observed.first.current.storedValue, "closed");
  await act(async () => observed.first.current.setValue("expanded"));
  assert.equal(window.localStorage.getItem("workspace-a"), '"open"');
  assert.equal(window.localStorage.getItem("workspace-b"), '"expanded"');
  await act(async () => {
    window.localStorage.setItem("workspace-a", '"collapsed"');
    window.dispatchEvent(new Event("local-storage:workspace-a"));
  });
  assert.equal(observed.first.current.storedValue, "expanded");
  await env.render(React.createElement(Preference, { storageKey: "workspace-c", fallback: "new-default", observed }));
  assert.equal(observed.first.current.storedValue, "new-default");
});

void test("successful writes publish the serialized JSON value", async (t) => {
  const env = await setup(t);
  const observed = await renderLinkedPreferences(env);
  await act(async () => observed.first.current.setValue({ active: true, omitted: undefined }));
  assert.equal(window.localStorage.getItem("panel"), '{"active":true}');
  assert.deepEqual(observed.first.current.storedValue, { active: true });
  assert.deepEqual(observed.second.current.storedValue, { active: true });
});

void test("callbacks captured for an old key cannot reset the current transient preference", async (t) => {
  const env = await setup(t);
  Object.defineProperty(window, "localStorage", { configurable: true, value: readOnlyStorage({}) });
  const observed = createPreferenceRefs();
  await env.render(React.createElement(Preference, { storageKey: "workspace-a", observed }));
  const previous = observed.first.current;
  await env.render(React.createElement(Preference, { storageKey: "workspace-b", observed }));
  await act(async () => observed.first.current.setValue("temporary-b"));
  assert.equal(observed.first.current.storedValue, "temporary-b");
  await act(async () => previous.setValue("late-a"));
  assert.equal(observed.first.current.storedValue, "temporary-b");
  await act(async () => previous.clearValue());
  assert.equal(observed.first.current.storedValue, "temporary-b");
});

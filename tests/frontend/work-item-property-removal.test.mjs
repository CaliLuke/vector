import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";
import { waitForDom } from "./helpers/wait-for-dom.mjs";
import { registerWorkItemTypesFormHarness } from "./helpers/work-item-types-form-harness.mjs";
registerWorkItemTypesFormHarness({
  sourceFiles: ["apps/web/core/components/work-item-types/property-removal-dialog.tsx"],
  reactGlobal: "propertyRemovalReact",
  react: React,
  stubs: {
    "@plane/propel/button": `const React=globalThis.propertyRemovalReact;export const Button=({variant,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
    "@/services/work-item-types.service": `export const typeErrorMessage=error=>error?.message??"Request failed";`,
    "./controls": `const React=globalThis.propertyRemovalReact;export const inputStyle="";export const Field=({label,children})=>React.createElement("label",null,label,children(label));export const PreviewDialog=({title,children,footer,onClose})=>React.createElement("section",{"aria-label":title},children,footer,React.createElement("button",{onClick:onClose},"Close"));`,
  },
});
const { PropertyRemovalDialog } =
  await import("../../apps/web/core/components/work-item-types/property-removal-dialog.tsx");
const button = (env, name) => [...env.container.querySelectorAll("button")].find((node) => node.textContent === name);
const waitForImpact = (env) =>
  act(async () =>
    waitForDom(
      env.container,
      () => /\d+ work items, \d+ types, and \d+ projects/.test(env.container.textContent),
      "Property review did not render its authoritative impact"
    )
  );
const deferred = () => {
  let resolve;
  const promise = new Promise((done) => {
    resolve = done;
  });
  return { promise, resolve };
};
async function screen(t, persistence, props = {}) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(PropertyRemovalDialog, {
        id: "risk",
        name: "Risk",
        revision: 7,
        persistence: { scope: "workspace", accountId: "owner", refreshRevision: async () => 7, ...persistence },
        onClose: () => {},
        ...props,
      })
    )
  );
  return env;
}
await test("removal waits for authoritative impact and retries the same receipt", async (t) => {
  const response = deferred();
  const requests = [];
  let attempts = 0;
  const env = await screen(t, {
    preview: async () => response.promise,
    commit: async (input) => {
      requests.push(input);
      if (++attempts === 1) throw new Error("Connection lost");
    },
  });
  assert.equal(button(env, "Delete property").disabled, true);
  assert.doesNotMatch(env.container.textContent, /0 work items/);
  await act(async () => response.resolve({ id: "review-one", items: 12, types: 2, projects: 3 }));
  await waitForImpact(env);
  assert.match(env.container.textContent, /12 work items, 2 types, and 3 projects/);
  await submitForm(env.container);
  assert.match(env.container.textContent, /Connection lost/);
  assert.equal(button(env, "Refresh review"), undefined);
  assert.ok(button(env, "Retry removal"));
  assert.equal(env.container.querySelector("fieldset").disabled, true);
  await submitForm(env.container);
  assert.equal(requests.length, 2);
  assert.deepEqual(requests[0], requests[1]);
  assert.equal(requests[0].review_id, "review-one");
});
await test("discarding current values requires a new review and exact confirmation", async (t) => {
  const reviews = [];
  const commits = [];
  const env = await screen(t, {
    preview: async (input) => {
      reviews.push(input);
      return { id: input.permanent ? "discard-review" : "preserve-review", items: 4, types: 1, projects: 2 };
    },
    commit: async (input) => {
      commits.push(input);
    },
  });
  await waitForImpact(env);
  await act(async () => env.container.querySelectorAll('input[type="radio"]')[1].click());
  await waitForImpact(env);
  assert.equal(reviews.at(-1).permanent, true);
  assert.match(env.container.textContent, /Existing history remains available/);
  await submitForm(env.container);
  assert.equal(commits.length, 0);
  await changeFormControl(env.container.querySelector('input[autocomplete="off"]'), "confirm");
  await submitForm(env.container);
  assert.equal(commits.length, 0);
  await changeFormControl(env.container.querySelector('input[autocomplete="off"]'), "Confirm");
  await submitForm(env.container);
  assert.equal(commits.length, 1);
  assert.equal(commits[0].review_id, "discard-review");
  assert.equal(commits[0].permanent, true);
});
await test("stale commit refreshes configuration and obtains a new review", async (t) => {
  const reviews = [];
  const commits = [];
  const env = await screen(t, {
    preview: async (input) => {
      reviews.push(input.revision);
      return { id: `review-${input.revision}`, items: 1, types: 1, projects: 1 };
    },
    refreshRevision: async () => 8,
    commit: async (input) => {
      commits.push(input);
      if (input.revision === 7) throw Object.assign(new Error("Configuration changed"), { response: { status: 409 } });
    },
  });
  await waitForImpact(env);
  await submitForm(env.container);
  await act(async () => button(env, "Refresh review").click());
  await waitForImpact(env);
  assert.deepEqual(reviews, [7, 8]);
  await submitForm(env.container);
  assert.equal(commits.at(-1).review_id, "review-8");
  assert.equal(commits.at(-1).revision, 8);
});

const revisedDefinition = {
  id: "risk",
  name: "risk",
  title: "Risk",
  description: "",
  icon: "",
  kind: "Cascading",
  configuration: {
    active: true,
    required: false,
    multiple: false,
    format: "",
    content: "",
    expression: "",
    result: "",
    default_value: null,
    options: [],
    levels: [
      { id: "a", name: "A", required: false },
      { id: "b", name: "B", required: false },
    ],
  },
  contexts: [],
};
await test("reviewed updates require Confirm for affected values and retry the immutable definition", async (t) => {
  const commits = [];
  let saved = 0;
  const env = await screen(
    t,
    {
      preview: async (input) => {
        assert.equal(input.action, "update");
        assert.deepEqual(input.definition, revisedDefinition);
        return { id: "update-review", items: 2, types: 1, projects: 1 };
      },
      commit: async (input) => {
        commits.push(structuredClone(input));
        if (commits.length === 1) throw new Error("Response lost");
      },
    },
    {
      definition: revisedDefinition,
      onCommitted: () => {
        saved++;
      },
    }
  );
  await waitForImpact(env);
  assert.equal(env.container.querySelectorAll('input[type="radio"]').length, 0);
  await submitForm(env.container);
  assert.equal(commits.length, 0);
  await changeFormControl(env.container.querySelector('input[autocomplete="off"]'), "confirm");
  await submitForm(env.container);
  assert.equal(commits.length, 0);
  await changeFormControl(env.container.querySelector('input[autocomplete="off"]'), "Confirm");
  await submitForm(env.container);
  assert.equal(saved, 0);
  assert.equal(button(env, "Refresh review"), undefined);
  assert.ok(button(env, "Retry changes"));
  await submitForm(env.container);
  assert.equal(saved, 1);
  assert.equal(commits.length, 2);
  assert.deepEqual(commits[0], commits[1]);
  assert.equal(commits[1].review_id, "update-review");
});
await test("an outdated full-definition edit cannot refresh its revision and overwrite newer settings", async (t) => {
  const revisions = [];
  const commits = [];
  const env = await screen(
    t,
    {
      preview: async (input) => {
        revisions.push(input.revision);
        return { id: "stale-update", items: 0, types: 1, projects: 0 };
      },
      refreshRevision: async () => 8,
      commit: async (input) => {
        commits.push(input);
        throw Object.assign(new Error("Configuration changed"), { response: { status: 409 } });
      },
    },
    { definition: revisedDefinition }
  );
  await waitForImpact(env);
  await submitForm(env.container);
  await act(async () => button(env, "Refresh review").click());
  assert.match(env.container.textContent, /Close and reopen this property/);
  assert.deepEqual(revisions, [7]);
  assert.equal(commits.length, 1);
  assert.equal(commits[0].revision, 7);
});

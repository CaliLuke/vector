import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { registerWorkItemTypesFormHarness } from "./helpers/work-item-types-form-harness.mjs";

const modalCoreUrl = new URL("../../packages/ui/src/modals/modal-core.tsx", import.meta.url).href;
const modalConstantsUrl = new URL("../../packages/ui/src/modals/constants.ts", import.meta.url).href;
registerWorkItemTypesFormHarness({
  sourceFiles: [
    "apps/web/core/components/work-item-types/workspace-hierarchy.tsx",
    "apps/web/core/components/work-item-types/controls.tsx",
  ],
  reactGlobal: "durableHierarchyReact",
  react: React,
  stubs: {
    "@plane/propel/button": `const React=globalThis.durableHierarchyReact;export const Button=({variant,prependIcon,children,...props})=>React.createElement("button",{type:"button",...props},children);`,
    "@plane/ui": `export { ModalCore } from ${JSON.stringify(modalCoreUrl)};export { EModalWidth } from ${JSON.stringify(modalConstantsUrl)};`,
    "@tanstack/react-router": `const React=globalThis.durableHierarchyReact;export const useParams=()=>({workspaceSlug:"types"});export const Link=({to,params,search,children,...props})=>React.createElement("a",{...props,href:to},children);`,
    "@/hooks/use-current-user": `export const useCurrentUserId=()=>"owner";`,
    "@/services/work-item-types.service": `export const typeErrorMessage=error=>error?.message??"Request failed";`,
    "@/services/work-item-hierarchy.service": `export const workItemHierarchyService={preview:(...args)=>globalThis.durableHierarchyPreview(...args)};`,
    "./type-icons": `export const typeIconComponents={task:()=>null};`,
  },
});
let HierarchyEditor;
const initial = () => ({
  enabled: true,
  can_manage: true,
  revision: 4,
  hierarchy_enabled: false,
  default_level: 0,
  types: [
    { id: "task", name: "Task", icon: "task", color: "#123456", level: 0 },
    { id: "bug", name: "Bug", icon: "task", color: "#123456", level: 0 },
  ],
});
const button = (name, root = document.body) =>
  [...root.querySelectorAll("button")].find((node) => node.textContent === name);
const click = (node) => act(async () => node.click());
const settle = () =>
  act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 10));
  });
async function screen(t, { catalog = initial(), preview, commit } = {}) {
  const env = await createReactDomTestEnvironment();
  ({ HierarchyEditor } = await import("../../apps/web/core/components/work-item-types/workspace-hierarchy.tsx"));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const reviews = [],
    commits = [];
  const harnessRef = React.createRef();
  globalThis.durableHierarchyPreview = async (_slug, input, signal) => {
    reviews.push(structuredClone(input));
    return preview
      ? preview(input, signal)
      : { id: `review-${reviews.length}`, links: 0, restricted_links: 0, pairs: [] };
  };
  function Harness() {
    const [current, setCurrent] = React.useState(catalog);
    const [pending, setPending] = React.useState(false);
    React.useImperativeHandle(harnessRef, () => ({ changeCatalog: setCurrent }), []);
    return React.createElement(HierarchyEditor, {
      catalog: current,
      pending,
      workspaceSlug: "types",
      accountId: "owner",
      onCommit: async (input) => {
        commits.push(structuredClone(input));
        setPending(true);
        try {
          if (commit) await commit(input);
          const next = {
            ...current,
            revision: current.revision + 1,
            hierarchy_enabled: true,
            default_level: input.default_level,
            types: structuredClone(current.types).map((type) => {
              type.level = input.placements.find((row) => row.type_id === type.id)?.level ?? 0;
              return type;
            }),
          };
          setCurrent(next);
          return next;
        } finally {
          setPending(false);
        }
      },
    });
  }
  await env.render(React.createElement(QueryClientProvider, { client }, React.createElement(Harness)));
  t.after(async () => {
    await env.unmount();
    client.clear();
  });
  return {
    env,
    client,
    reviews,
    commits,
    changeCatalog: (next) => act(async () => harnessRef.current.changeCatalog(next)),
    move: (level) =>
      changeFormControl(env.container.querySelector('select[aria-label="Move Bug to level"]'), String(level)),
  };
}
await test("first all-zero hierarchy activation stays local until its reviewed save", async (t) => {
  const { env, reviews, commits } = await screen(t);
  await click(button("Enable hierarchy"));
  assert.equal(reviews.length, 0);
  assert.equal(commits.length, 0);
  assert.equal(button("Save changes").disabled, false);
  await click(button("Discard"));
  assert.ok(button("Enable hierarchy"));
  await click(button("Enable hierarchy"));
  await click(button("Save changes"));
  await settle();
  assert.equal(reviews.length, 1);
  assert.equal(commits.length, 0);
  assert.equal(document.body.querySelector('input[autocomplete="off"]'), null);
  await click(button("Save changes", document.body.querySelector('[role="dialog"]')));
  await settle();
  assert.equal(commits[0].revision, 4);
  assert.equal(commits[0].review_id, "review-1");
  assert.equal(button("Save changes", env.container).disabled, true);
});
await test("cancel retains placements and reopening obtains a fresh impact receipt", async (t) => {
  const { env, reviews, move } = await screen(t, {
    preview: async () => ({ id: "private-review", links: 3, restricted_links: 3, pairs: [] }),
  });
  await click(button("Enable hierarchy"));
  await move(1);
  await click(button("Save changes"));
  await settle();
  assert.match(document.body.textContent, /3 additional links belong to private drafts/);
  await click(button("Go back and fix"));
  assert.equal(env.container.querySelector('select[aria-label="Move Bug to level"]').value, "1");
  await click(button("Save changes"));
  await settle();
  assert.equal(reviews.length, 2);
});
await test("ambiguous commit retries its immutable receipt with exact lowercase confirmation", async (t) => {
  let attempts = 0;
  const { reviews, commits, move, changeCatalog } = await screen(t, {
    preview: async () => ({ id: "receipt", links: 2, restricted_links: 2, pairs: [] }),
    commit: async () => {
      if (++attempts === 1) throw new Error("Connection lost");
    },
  });
  await click(button("Enable hierarchy"));
  await move(1);
  await click(button("Save changes"));
  await settle();
  const input = document.body.querySelector('input[autocomplete="off"]');
  await changeFormControl(input, "Confirm");
  await click(button("Save anyway"));
  assert.equal(commits.length, 0);
  await changeFormControl(input, "confirm");
  await click(button("Save anyway"));
  await settle();
  assert.match(document.body.textContent, /Connection lost/);
  await changeCatalog({ ...initial(), revision: 5, hierarchy_enabled: true });
  await click(button("Save anyway"));
  await settle();
  assert.equal(reviews.length, 1);
  assert.equal(commits.length, 2);
  assert.deepEqual(commits[1], commits[0]);
});
await test("a remotely changed catalog retains draft and requires explicit discard before a new review", async (t) => {
  const { env, move, changeCatalog } = await screen(t, { catalog: { ...initial(), hierarchy_enabled: true } });
  await move(1);
  await changeCatalog({ ...initial(), hierarchy_enabled: true, revision: 5 });
  assert.equal(env.container.querySelector('select[aria-label="Move Bug to level"]').value, "1");
  assert.equal(button("Save changes").disabled, true);
  await click(button("Discard and reload"));
  assert.equal(env.container.querySelector('select[aria-label="Move Bug to level"]').value, "0");
  assert.equal(button("Discard and reload"), undefined);
});
await test("preview failures refresh the scoped catalog while leaving the draft intact", async (t) => {
  const { client, env, move } = await screen(t, {
    preview: async () => {
      throw new Error("Configuration changed");
    },
  });
  client.setQueryData(["workspace", "types", "hierarchy", "owner"], initial());
  await click(button("Enable hierarchy"));
  await move(1);
  await click(button("Save changes"));
  await settle();
  assert.equal(client.getQueryState(["workspace", "types", "hierarchy", "owner"]).isInvalidated, true);
  await click(button("Go back and fix"));
  assert.equal(env.container.querySelector('select[aria-label="Move Bug to level"]').value, "1");
});

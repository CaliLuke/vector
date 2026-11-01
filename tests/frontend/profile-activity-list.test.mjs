import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

globalThis.profileActivityList = { React, editors: [] };
registerWebTestModules({
  "@/hooks/use-current-user": 'export const useCurrentUser=()=>({data:{id:"self"}});',
  "@/hooks/use-workspaces": 'export const useWorkspaceBySlug=()=>({data:{id:"workspace-id"}});',
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
  "@/hooks/use-labels": "export const useWorkspaceLabels=()=>({getWorkspaceLabels:()=>[]});",
  "@/components/editor/rich-text": `export const RichTextEditor=props=>{
    const state=globalThis.profileActivityList; state.editors.push(props);
    return state.React.createElement("output", {"data-editor-id":props.id}, props.initialValue);
  };`,
});
const { ActivityList } = await import("../../apps/web/core/components/profile/activity/activity-list.tsx");

function entry(id, overrides = {}) {
  return {
    id,
    field: "name",
    verb: "updated",
    new_value: "New name",
    old_value: "Old name",
    created_at: "2026-10-31T12:00:00Z",
    project: "project-id",
    issue: "issue-id",
    project_detail: { identifier: "PRJ" },
    issue_detail: { name: "Work item", sequence_id: 1 },
    workspace_detail: { slug: "workspace" },
    actor_detail: { id: "self", display_name: "Person", is_bot: false },
    ...overrides,
  };
}

void test("profile activity preserves actor attribution, comment scope and hidden metadata changes", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const results = [
    entry("self"),
    entry("archive", { field: "archived_at", new_value: "archive" }),
    entry("bot", { actor_detail: { id: "bot", is_bot: true, first_name: "Automation" } }),
    entry("comment", { field: "comment", new_value: "", old_value: "Retained comment" }),
    entry("internal", { field: "updated_by" }),
  ];
  await createRouterFixture("/workspace/profile/self/").render(
    env,
    React.createElement(ActivityList, { activity: { results } })
  );
  const list = env.container.querySelector("ul");
  assert.equal(list.children.length, 4);
  assert.ok([...list.children].every((child) => child.tagName === "LI"));
  assert.match(list.children[0].textContent, /You/);
  assert.equal(list.children[0].querySelector('a[href="/workspace/profile/self/"]')?.textContent, "You");
  assert.match(list.children[1].textContent, /Vector/);
  assert.match(list.children[2].textContent, /Automation Bot/);
  assert.equal(list.querySelector('[data-editor-id="comment"]').textContent, "Retained comment");
  const props = globalThis.profileActivityList.editors.at(-1);
  assert.equal(props.editable, false);
  assert.equal(props.workspaceSlug, "workspace");
  assert.equal(props.workspaceId, "workspace-id");
  assert.equal(props.projectId, "project-id");
});

void test("unknown activity fields cannot resolve object prototype properties as renderers", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const results = ["constructor", "__proto__", "toString"].map((field) => entry(field, { field }));
  await createRouterFixture("/workspace/profile/self/").render(
    env,
    React.createElement(ActivityList, { activity: { results } })
  );
  assert.equal(env.container.querySelectorAll("ul > li").length, 3);
});

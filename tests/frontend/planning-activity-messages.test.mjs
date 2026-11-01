import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebTestModules, reactModuleUrl } from "./helpers/web-test-modules.mjs";

const block = `import React from ${JSON.stringify(reactModuleUrl)}; export const IssueActivityBlockComponent=({children,icon,ends})=>React.createElement('section',{'data-ends':ends},icon,children);`;
registerWebTestModules({}, (specifier, context, next) => {
  if (
    (specifier === "./helpers/activity-block" || specifier === "./") &&
    /\/(planning|cycle|module|relation)\.tsx$/.test(context.parentURL ?? "")
  ) {
    return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(block)}` };
  }
  return next(specifier, context);
});

const components = {
  cycle: (await import("../../apps/web/core/components/issues/issue-detail/issue-activity/activity/actions/cycle.tsx"))
    .IssueCycleActivity,
  module: (
    await import("../../apps/web/core/components/issues/issue-detail/issue-activity/activity/actions/module.tsx")
  ).IssueModuleActivity,
};
for (const [kind, Component] of Object.entries(components)) {
  for (const verb of ["created", "updated", "deleted"]) {
    void test(`${kind} history retains its ${verb} label and scoped link`, async (t) => {
      const env = await createReactDomTestEnvironment();
      t.after(() => env.unmount());
      await env.render(
        React.createElement(Component, {
          activity: {
            verb,
            workspace_detail: { slug: "workspace" },
            project: "project",
            new_identifier: "new",
            old_identifier: "old",
            new_value: "New <name>",
            old_value: "Retired <name>",
          },
          ends: "top",
        })
      );
      const link = env.container.querySelector("a");
      assert.equal(
        link.getAttribute("href"),
        `/workspace/projects/project/${kind}s/${verb === "deleted" ? "old" : "new"}`
      );
      assert.equal(link.textContent.trim(), verb === "deleted" ? "Retired <name>" : "New <name>");
      assert.equal(link.querySelector("name"), null);
      assert.equal(link.target, "_blank");
      assert.equal(link.rel, "noopener noreferrer");
      const expected =
        verb === "created"
          ? `added this work item to the ${kind} `
          : verb === "updated"
            ? `set the ${kind} to `
            : `removed the work item from the ${kind} `;
      assert.ok(env.container.textContent.startsWith(expected));
      await env.render(React.createElement(Component, { activity: null }));
      assert.equal(env.container.textContent, "");
    });
  }
}

const { IssueRelationActivity } =
  await import("../../apps/web/core/components/issues/issue-detail/issue-activity/activity/actions/relation.tsx");
for (const [field, added, removed] of [
  ["blocking", "marked this work item is blocking work item ", "removed the blocking work item "],
  ["blocked_by", "marked this work item is being blocked by ", "removed this work item being blocked by work item "],
  ["duplicate", "marked this work item as duplicate of ", "removed this work item as a duplicate of "],
  ["relates_to", "marked that this work item relates to ", "removed the relation from "],
]) {
  for (const deleting of [false, true]) {
    void test(`${field} history preserves the ${deleting ? "removed" : "added"} relation snapshot`, async (t) => {
      const env = await createReactDomTestEnvironment();
      t.after(() => env.unmount());
      await env.render(
        React.createElement(IssueRelationActivity, {
          activity: { field, old_value: deleting ? "Old <item>" : "", new_value: "New <item>" },
          ends: "bottom",
        })
      );
      assert.equal(
        env.container.textContent,
        (deleting ? removed : added) + (deleting ? "Old <item>" : "New <item>") + "."
      );
      assert.equal(env.container.querySelector("item"), null);
      assert.ok(env.container.querySelector("svg"));
      assert.equal(env.container.querySelector("section").dataset.ends, "bottom");
    });
  }
}

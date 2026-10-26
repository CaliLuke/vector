import assert from "node:assert/strict";
import { test } from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { createPreviewContent } = await import("../../apps/web/core/components/work-item-types/preview-content.ts");
const {
  preserveProperties,
  effectiveProperty,
  contextOverlap,
  hierarchyConflicts,
  removedCascadeValues,
  defaultPropertyValues,
} = await import("../../apps/web/core/components/work-item-types/preview-actions.ts");

await test("removing a used property preserves readable values without duplicating them on retry", () => {
  const content = createPreviewContent();
  const severity = content.properties.find((candidate) => candidate.id === "severity");
  const original = content.items[0];
  const changed = preserveProperties(original, [severity], "Type changed");
  assert.ok(changed.description.startsWith(original.description));
  assert.match(changed.description, /Severity: Major/);
  assert.equal(changed.values.severity, undefined);
  assert.equal(changed.values.effort, "3");
  assert.deepEqual(preserveProperties(changed, [severity], "Type changed"), changed);
  assert.equal(original.values.severity, "major");
});

await test("preservation does not mistake false and zero for missing values", () => {
  const content = createPreviewContent();
  const item = { ...content.items[0], values: { blocking: false, effort: "0" } };
  const changed = preserveProperties(item, content.properties, "Properties removed");
  assert.match(changed.description, /Release blocker: No/);
  assert.match(changed.description, /Effort: 0/);
});

await test("a scoped optional override changes only the matching project's field", () => {
  const property = createPreviewContent().properties.find((candidate) => candidate.id === "severity");
  assert.equal(effectiveProperty(property, "mobile", "bug").required, false);
  assert.equal(effectiveProperty(property, "platform", "bug").required, true);
  assert.equal(effectiveProperty(property, "mobile", "incident").required, true);
  assert.equal(property.required, true);
});

await test("context overlap catches project-only versus type-only ambiguity", () => {
  assert.equal(contextOverlap({ projects: ["mobile"], types: [] }, { projects: [], types: ["bug"] }), true);
  assert.equal(
    contextOverlap({ projects: ["mobile"], types: ["bug"] }, { projects: ["platform"], types: ["bug"] }),
    false
  );
});

await test("hierarchy preview identifies only links broken by moving a type", () => {
  const content = createPreviewContent();
  const levels = Object.fromEntries(content.types.map((type) => [type.id, type.level]));
  assert.deepEqual(hierarchyConflicts(content, levels), []);
  const conflicts = hierarchyConflicts(content, { ...levels, task: 1 });
  assert.deepEqual(
    conflicts.map((item) => item.id),
    ["WEB-143"]
  );
  assert.equal(content.items.find((item) => item.id === "WEB-143").parent, "WEB-142");
});

await test("defaults use the selected context without supplying required or hidden fields", () => {
  const content = createPreviewContent();
  const severity = content.properties.find((candidate) => candidate.id === "severity");
  severity.defaultValue = "major";
  severity.contexts[0].defaultValue = "minor";
  assert.equal(defaultPropertyValues(content, "platform", "bug").severity, undefined);
  assert.equal(defaultPropertyValues(content, "mobile", "bug").severity, "minor");
  severity.contexts[0].active = false;
  assert.equal(defaultPropertyValues(content, "mobile", "bug").severity, undefined);
});

await test("removing cascading options affects only work items using that context", () => {
  const content = createPreviewContent();
  const property = content.properties.find((candidate) => candidate.kind === "Cascading");
  const scoped = {
    ...property,
    contexts: [
      {
        id: "mobile",
        name: "Mobile",
        projects: ["mobile"],
        types: [],
        required: null,
        active: null,
        options: [{ id: "mobile-ui", name: "Mobile UI", level: 0, parent: "", nextLabel: "" }],
      },
    ],
  };
  content.items.push({ ...content.items[0], id: "MOB-1", project: "mobile", values: { [property.id]: ["mobile-ui"] } });
  const changed = { ...scoped, contexts: [{ ...scoped.contexts[0], options: [] }] };
  assert.deepEqual(
    removedCascadeValues(content, scoped, changed).map((item) => item.id),
    ["MOB-1"]
  );
  const preserved = preserveProperties(
    content.items.at(-1),
    [effectiveProperty(scoped, "mobile", "bug")],
    "Options removed"
  );
  assert.match(preserved.description, /Mobile UI/);
});

await test("removed rich text remains readable without exposing editor HTML", () => {
  const content = createPreviewContent();
  const property = content.properties.find((candidate) => candidate.kind === "Rich text");
  const item = {
    ...content.items[0],
    values: { [property.id]: "<ol><li><p>Sign in.</p></li><li><p>Open another device.</p></li></ol>" },
  };
  const changed = preserveProperties(item, [property], "Type changed");
  assert.match(changed.description, /1\. Sign in\./);
  assert.match(changed.description, /2\. Open another device\./);
  assert.doesNotMatch(changed.description, /<\/?(?:ol|li|p)>/);
});

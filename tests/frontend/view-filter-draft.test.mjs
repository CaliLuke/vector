import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { createFilterDraft, filterDraftExpression } =
  await import("../../apps/web/core/components/views/filter-draft.ts");

await test("filter drafts preserve nested logic, ordering, duplicate conditions and literal wire values", () => {
  const expression = {
    and: [
      { priority__in: "high,urgent" },
      { or: [{ customproperty_count__gte: 0 }, { not: { state_group__in: ["completed"] } }] },
      { customproperty_flag__exact: false },
      { customproperty_missing__exact: null },
      { customproperty_values__in: [0, 1] },
      { priority__in: "high,urgent" },
    ],
  };
  assert.deepEqual(filterDraftExpression(createFilterDraft(expression)), expression);
});

await test("filter drafts retain empty filters and empty values", () => {
  assert.deepEqual(filterDraftExpression(createFilterDraft({})), {});
  assert.deepEqual(filterDraftExpression(createFilterDraft({ label_id__in: [] })), { label_id__in: [] });
  assert.deepEqual(filterDraftExpression(createFilterDraft({ customproperty_text__exact: "" })), {
    customproperty_text__exact: "",
  });
});

await test("filter drafts preserve all five API node levels", () => {
  const expression = { and: [{ not: { or: [{ and: [{ priority__exact: "high" }] }] } }] };
  assert.deepEqual(filterDraftExpression(createFilterDraft(expression)), expression);
});

await test("invalid filter references cannot silently drop a condition", () => {
  assert.throws(() => filterDraftExpression([{ kind: "not", child: 0 }]), /missing child/);
});

await test("malformed sibling predicates survive conversion for the editor's explicit repair flow", () => {
  const expression = {
    and: [{ priority__exact: "high", not: { unknown__exact: "nested" } }],
    or: [{ label_id__in: ["label"] }],
    unsupported__exact: "kept",
  };
  assert.deepEqual(filterDraftExpression(createFilterDraft(expression)), expression);
});

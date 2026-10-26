import assert from "node:assert/strict";
import { act } from "./react.mjs";
import { state } from "./issue-form-harness.mjs";

export function propertyDefinition(id, kind, configuration = {}) {
  return {
    id,
    name: id,
    title: id,
    description: "",
    icon: "",
    kind,
    context_id: "",
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
      levels: [],
      ...configuration,
    },
  };
}
export function enableTypeFields(properties) {
  state.types = {
    enabled: true,
    types: ["task", "bug"].map((id) => ({ id, name: id, icon: id, color: "#ef4444", active: true, available: true })),
  };
  state.schema = async (input) => ({
    enabled: true,
    type_id: input.type_id || "task",
    eligible_type_ids: ["task", "bug"],
    selection_reason: "",
    properties: typeof properties === "function" ? properties(input) : properties,
    members: [],
    releases: [],
    computed: {},
  });
}
export function customControl(container, title) {
  const legend = [...container.querySelectorAll("legend")].find((node) => node.textContent.trim() === title);
  assert.ok(legend, `field ${title} is visible`);
  return legend.parentElement.querySelector("input, textarea, button, select");
}
export async function chooseWorkItemType(container, type) {
  await act(async () => container.querySelector('[aria-label="Work item type"]').click());
  const option = [...document.querySelectorAll('[role="option"]')].find((node) => node.textContent === type);
  assert.ok(option);
  await act(async () => {
    option.dispatchEvent(new MouseEvent("mousedown", { button: 0, bubbles: true, cancelable: true }));
    option.click();
  });
}

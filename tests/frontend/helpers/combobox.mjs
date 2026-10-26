import assert from "node:assert/strict";
import { clickWithPointer } from "./pointer.mjs";

export async function openSearchCombobox(trigger) {
  assert.ok(trigger, "the picker trigger should render");
  await clickWithPointer(trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "true", "native combobox state should be open");
  const input = document.querySelector('input[role="combobox"]');
  assert.ok(input, "the search input should mount");
  assert.equal(document.activeElement, input, "opening the picker should focus its search input");
  return input;
}

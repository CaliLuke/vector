import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { toFilterArray, getFilterValueLength } = await import("../../packages/utils/src/rich-filters/values/core.ts");
const { getSingleSelectConfig, getMultiSelectConfig } =
  await import("../../packages/utils/src/rich-filters/factories/configs/core.ts");

void test("filter arrays retain valid falsy values and stable array identity while removing absent values", () => {
  const date = new Date(0);
  const values = ["", false, 0, date];
  assert.equal(toFilterArray(values), values);
  assert.deepEqual(toFilterArray([null, ...values, undefined]), values);
  assert.deepEqual(toFilterArray(null), []);
  assert.deepEqual(toFilterArray(undefined), []);
  assert.deepEqual(toFilterArray(false), [false]);
  assert.deepEqual(toFilterArray(0), [0]);
  assert.equal(toFilterArray(date)[0], date);
});

void test("filter value counts distinguish absent values from falsy selections", () => {
  assert.equal(getFilterValueLength(null), 0);
  assert.equal(getFilterValueLength(undefined), 0);
  assert.equal(getFilterValueLength([]), 0);
  assert.equal(getFilterValueLength([0, false]), 2);
  assert.equal(getFilterValueLength(false), 1);
  assert.equal(getFilterValueLength(0), 1);
  assert.equal(getFilterValueLength(""), 1);
});

void test("single and multi select options retain typed icon data and selection values", () => {
  const item = { id: "one", label: "First", selected: false, color: "red" };
  const transforms = {
    items: [item],
    getId: (row) => row.id,
    getLabel: (row) => row.label,
    getValue: (row) => row.selected,
    getIconData: (row) => row.color,
  };
  const icons = { getOptionIcon: (color) => `icon:${color}` };
  const single = getSingleSelectConfig(transforms, { isOperatorEnabled: true }, icons);
  const multi = getMultiSelectConfig(transforms, { isOperatorEnabled: true, singleValueOperator: "in" }, icons);
  const expected = [{ id: "one", label: "First", value: false, icon: "icon:red" }];
  assert.deepEqual(single.getOptions(), expected);
  assert.deepEqual(multi.getOptions(), expected);
});

void test("selection factories never invoke an icon renderer without its required data transform", () => {
  const transforms = {
    items: ["one"],
    getId: (value) => value,
    getLabel: (value) => value,
    getValue: (value) => value,
  };
  const config = getSingleSelectConfig(
    transforms,
    { isOperatorEnabled: true },
    {
      getOptionIcon: () => {
        throw new Error("missing icon data");
      },
    }
  );
  assert.deepEqual(config.getOptions(), [{ id: "one", label: "one", value: "one", icon: undefined }]);
});

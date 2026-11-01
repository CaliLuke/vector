import * as ContractEnums from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import {
  buildTree,
  checkDuplicates,
  checkIfArraysHaveSameElements,
  getValidKeysFromObject,
  convertStringArrayToBooleanObject,
  findStringWithMostCharacters,
  groupBy,
  groupByField,
  orderArrayBy,
  orderGroupedDataByField,
  sortByCurrentUserThenSelected,
  sortByField,
  sortBySelectedFirst,
} from "../../packages/utils/src/array.ts";

registerWebModuleHooks();

await test("field grouping treats prototype-like names as ordinary own groups", () => {
  const first = { category: "__proto__", id: "first" };
  const second = { category: "__proto__", id: "second" };
  const constructor = { category: "constructor", id: "constructor" };
  const groups = groupByField([first, constructor, second], "category");
  assert.deepEqual(groups.__proto__, [first, second]);
  assert.deepEqual(groups.constructor, [constructor]);
  assert.equal(Object.hasOwn(groups, "__proto__"), true);
  assert.equal(Object.getPrototypeOf(groups), Object.prototype);
});

await test("group ordering handles own method-name groups and ignores inherited groups", () => {
  const inheritedRows = [{ order: 2 }, { order: 1 }];
  const groups = Object.assign(Object.create({ inherited: inheritedRows }), {
    hasOwnProperty: [{ order: 3 }, { order: 1 }],
  });
  assert.equal(orderGroupedDataByField(groups, "order"), groups);
  assert.deepEqual(groups.hasOwnProperty, [{ order: 1 }, { order: 3 }]);
  assert.deepEqual(inheritedRows, [{ order: 2 }, { order: 1 }]);
});

await test("string collection helpers retain special keys and stable longest-string ties", () => {
  const flags = convertStringArrayToBooleanObject(["__proto__", "constructor", "normal", "normal"]);
  assert.deepEqual(Object.keys(flags), ["__proto__", "constructor", "normal"]);
  assert.equal(flags.__proto__, true);
  assert.equal(flags.constructor, true);
  assert.equal(Object.getPrototypeOf(flags), Object.prototype);
  assert.equal(findStringWithMostCharacters([]), "");
  assert.equal(findStringWithMostCharacters(["a", "longest", "bb", "another"]), "longest");
});

await test("nested grouping and ordering preserve rows, missing groups and stable ties", () => {
  const a = { id: "a", data: { rank: 2, group: "first" } };
  const b = { id: "b", data: { rank: 1, group: "second" } };
  const c = { id: "c", data: { rank: 2, group: "first" } };
  const absent = { id: "absent", data: null };
  const rows = Object.freeze([a, b, c]);
  assert.deepEqual(orderArrayBy(rows, "data.rank"), [b, a, c]);
  assert.deepEqual(orderArrayBy(rows, "-data.rank"), [a, c, b]);
  assert.deepEqual(rows, [a, b, c]);
  assert.deepEqual(groupBy([...rows, absent], "data.group"), { first: [a, c], second: [b], None: [absent] });
  const special = { group: "__proto__" };
  assert.deepEqual(groupBy([special], "group")["__proto__"], [special]);
});

await test("in-place field ordering preserves the caller's arrays and stable ties", () => {
  const first = { id: "first", order: 2 };
  const second = { id: "second", order: 1 };
  const tied = { id: "tied", order: 2 };
  const rows = [first, second, tied];
  assert.equal(sortByField(rows, "order"), rows);
  assert.deepEqual(rows, [second, first, tied]);
  const groupRows = [first, second, tied];
  const groups = { group: groupRows };
  assert.equal(orderGroupedDataByField(groups, "order"), groups);
  assert.equal(groups.group, groupRows);
  assert.deepEqual(groupRows, rows);
});

await test("selection ordering leaves source options intact and keeps stable groups", () => {
  const options = Object.freeze([
    { value: "other" },
    { value: "selected" },
    { value: "current" },
    { value: null },
    { value: "last" },
  ]);
  assert.deepEqual(sortBySelectedFirst(options, ["selected", "current"]), [
    options[1],
    options[2],
    options[0],
    options[3],
    options[4],
  ]);
  assert.deepEqual(sortByCurrentUserThenSelected(options, ["selected"], "current"), [
    options[2],
    options[1],
    options[0],
    options[3],
    options[4],
  ]);
  assert.equal(sortBySelectedFirst(options, []), options);
  assert.equal(options[0].value, "other");
});

await test("label trees preserve row identity while attaching ordered children", () => {
  const root = { id: "root", parent: null };
  const child = { id: "child", parent: "root" };
  const sibling = { id: "sibling", parent: null };
  const tree = buildTree([root, child, sibling]);
  assert.deepEqual(
    tree.map((row) => row.id),
    ["root", "sibling"]
  );
  assert.equal(tree[0], root);
  assert.equal(root.children[0], child);
  assert.deepEqual(child.children, []);
  assert.deepEqual(sibling.children, []);
});

await test("collection inspection retains equality and own truthy-key semantics", () => {
  const reference = {};
  assert.equal(checkDuplicates([reference, reference]), true);
  assert.equal(checkDuplicates([{}, {}]), false);
  assert.equal(checkIfArraysHaveSameElements([reference, 1], [1, reference]), true);
  assert.equal(checkIfArraysHaveSameElements(null, []), false);
  const values = Object.assign(Object.create({ inherited: true }), { kept: 1, zero: 0, empty: "", nil: null });
  assert.deepEqual(getValidKeysFromObject(values), ["kept"]);
  assert.deepEqual(getValidKeysFromObject([1]), []);
  assert.deepEqual(getValidKeysFromObject(null), []);
});

void test("state, workspace and weekday presentation ordering preserves shared input arrays", async () => {
  const { sortStates } = await import("../../packages/utils/src/work-item/state.ts");
  const { orderWorkspacesList } = await import("../../packages/utils/src/workspace.ts");
  const { getOrderedDays } = await import("../../packages/utils/src/calendar.ts");
  const completed = { id: "completed", group: "completed", sequence: 0 };
  const later = { id: "later", group: "started", sequence: 2 };
  const first = { id: "first", group: "started", sequence: 1 };
  const states = Object.freeze([completed, later, first]);
  assert.deepEqual(sortStates(states), [first, later, completed]);
  assert.deepEqual(states, [completed, later, first]);
  assert.equal(sortStates([]), undefined);
  const zulu = { name: "Zulu" };
  const alpha = { name: "Alpha" };
  const workspaces = Object.freeze([zulu, alpha]);
  assert.deepEqual(orderWorkspacesList(workspaces), [alpha, zulu]);
  assert.deepEqual(workspaces, [zulu, alpha]);
  const days = Object.freeze([0, 1, 2, 3, 4, 5, 6]);
  assert.deepEqual(
    getOrderedDays(days, (day) => day, ContractEnums.EStartOfTheWeek.MONDAY),
    [1, 2, 3, 4, 5, 6, 0]
  );
  assert.deepEqual(days, [0, 1, 2, 3, 4, 5, 6]);
});

void test("work item counts follow layout grouping and cannot become NaN for missing groups", async () => {
  const { issueCountBasedOnFilters } = await import("../../packages/utils/src/work-item/base.ts");
  const { EIssueLayoutTypes: layout } = await import("../../packages/types/dist/index.mjs");
  const ids = ["first", "second"];
  const groups = { first: ids, second: ["second"], missing: undefined };
  const subgroups = { outer: groups, empty: undefined, other: { last: ["third"] } };
  assert.equal(issueCountBasedOnFilters(ids, layout.SPREADSHEET), 2);
  assert.equal(issueCountBasedOnFilters(ids, layout.GANTT), 2);
  assert.equal(issueCountBasedOnFilters(ids, layout.LIST), 2);
  assert.equal(issueCountBasedOnFilters(groups, layout.LIST, "state"), 3);
  assert.equal(issueCountBasedOnFilters(groups, layout.CALENDAR), 3);
  assert.equal(issueCountBasedOnFilters(groups, layout.KANBAN, "state"), 3);
  assert.equal(issueCountBasedOnFilters(subgroups, layout.KANBAN, "state", "assignee"), 4);
  assert.equal(issueCountBasedOnFilters(groups, layout.KANBAN), 0);
  assert.equal(issueCountBasedOnFilters(ids, layout.CALENDAR), 0);
  assert.equal(issueCountBasedOnFilters(groups, layout.SPREADSHEET), 0);
  assert.equal(issueCountBasedOnFilters(undefined, layout.LIST), 0);
  assert.equal(issueCountBasedOnFilters(ids, "unknown"), 0);
});

void test("page filters preserve public/private/archive distinctions and combine ownership, dates and favorites", async () => {
  const { filterPagesByPageType, shouldFilterPage, orderPages } = await import("../../packages/utils/src/page.ts");
  const { EPageAccess } = await import("../../packages/types/dist/index.mjs");
  const pages = [
    {
      id: "public",
      name: "Zulu",
      access: EPageAccess.PUBLIC,
      owned_by: "owner",
      is_favorite: true,
      created_at: "2026-10-01",
    },
    { id: "private", name: "Alpha", access: EPageAccess.PRIVATE, owned_by: "other", created_at: "2026-10-02" },
    { id: "archive", name: "Beta", access: EPageAccess.PUBLIC, archived_at: "2026-10-03" },
  ];
  assert.deepEqual(
    filterPagesByPageType("public", pages).map((page) => page.id),
    ["public"]
  );
  assert.deepEqual(
    filterPagesByPageType("private", pages).map((page) => page.id),
    ["private"]
  );
  assert.deepEqual(
    filterPagesByPageType("archived", pages).map((page) => page.id),
    ["archive"]
  );
  assert.equal(shouldFilterPage(pages[0], { created_by: ["owner"], favorites: true }), true);
  assert.equal(shouldFilterPage(pages[1], { created_by: ["owner"] }), false);
  assert.equal(shouldFilterPage(pages[1], { favorites: true }), false);
  assert.equal(shouldFilterPage({ ...pages[0], created_at: "invalid" }, { created_at: ["2026-10-01;after"] }), false);
  assert.deepEqual(
    orderPages(Object.freeze(pages), "name", "desc").map((page) => page.id),
    ["public", "archive", "private"]
  );
});

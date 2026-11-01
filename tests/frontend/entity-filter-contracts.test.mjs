import * as ContractEnums from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { shouldFilterView } = await import("../../packages/utils/src/project-views.ts");
const { shouldFilterProject } = await import("../../packages/utils/src/project.ts");
const { shouldFilterModule } = await import("../../packages/utils/src/module.ts");
const { shouldFilterCycle, getCycleGroup } = await import("../../packages/utils/src/cycle.ts");
const { calculateTotalFilters, matchesDateFilters, satisfiesDateFilter } =
  await import("../../packages/utils/src/filter.ts");
const range = ["2026-10-01;after", "2026-10-31;before"];

void test("view access cannot override an owner or date restriction, regardless of filter key order", () => {
  const view = {
    created_by: "owner",
    created_at: "2026-10-15",
    access: ContractEnums.EViewAccess.PUBLIC,
    is_favorite: true,
  };
  assert.equal(shouldFilterView(view, { owned_by: ["other"], view_type: [ContractEnums.EViewAccess.PUBLIC] }), false);
  assert.equal(shouldFilterView(view, { view_type: [ContractEnums.EViewAccess.PUBLIC], owned_by: ["other"] }), false);
  assert.equal(
    shouldFilterView(view, { created_at: ["2026-11-01;after"], view_type: [ContractEnums.EViewAccess.PUBLIC] }),
    false
  );
  assert.equal(
    shouldFilterView(view, {
      owned_by: ["owner"],
      view_type: [ContractEnums.EViewAccess.PUBLIC],
      created_at: range,
      favorites: true,
    }),
    true
  );
  assert.equal(shouldFilterView(view, { view_type: [ContractEnums.EViewAccess.PRIVATE] }), false);
  assert.equal(shouldFilterView({ ...view, is_favorite: false }, { favorites: true }), false);
  assert.equal(shouldFilterView(view, undefined), true);
  assert.equal(shouldFilterView({ ...view, is_favorite: undefined }, { favorites: true }), false);
});

void test("project filters match both expanded and scalar leads and preserve archive and membership constraints", () => {
  const project = {
    network: 2,
    project_lead: "lead",
    members: ["member"],
    created_at: "2026-10-15",
    member_role: ContractEnums.EUserPermissions.ADMIN,
  };
  const filters = { access: ["2"], lead: ["lead"], members: ["other", "member"], created_at: range };
  assert.equal(shouldFilterProject(project, { my_projects: true }, filters), true);
  assert.equal(shouldFilterProject({ ...project, project_lead: { id: "lead" } }, {}, filters), true);
  assert.equal(shouldFilterProject({ ...project, project_lead: null }, {}, filters), false);
  assert.equal(shouldFilterProject({ ...project, members: [] }, {}, filters), false);
  assert.equal(shouldFilterProject({ ...project, member_role: undefined }, { my_projects: true }, {}), false);
  assert.equal(shouldFilterProject(project, { archived_projects: true }, {}), false);
  assert.equal(shouldFilterProject({ ...project, archived_at: "2026-10-20" }, {}, {}), false);
  assert.equal(
    shouldFilterProject({ ...project, archived_at: "2026-10-20" }, { archived_projects: true }, filters),
    true
  );
});

void test("module and cycle filters require every selected field while accepting any member in a selection", () => {
  const module = {
    status: "STARTED",
    lead_id: "lead",
    member_ids: ["member"],
    start_date: "2026-10-01",
    target_date: "2026-10-31",
    is_favorite: true,
  };
  const filters = {
    status: ["started"],
    lead: ["lead"],
    members: ["member", "other"],
    start_date: range,
    target_date: range,
  };
  assert.equal(shouldFilterModule(module, { favorites: true }, filters), true);
  assert.equal(shouldFilterModule({ ...module, member_ids: undefined }, {}, filters), false);
  assert.equal(shouldFilterModule({ ...module, target_date: "2026-11-01" }, {}, filters), false);
  assert.equal(shouldFilterModule({ ...module, is_favorite: false }, { favorites: true }, filters), false);
  assert.equal(shouldFilterModule({}, {}, { status: [], members: null }), true);
  assert.equal(shouldFilterModule({ ...module, is_favorite: undefined }, { favorites: true }, filters), false);
  const cycle = { status: "CURRENT", start_date: "2026-10-01", end_date: "2026-10-31" };
  assert.equal(shouldFilterCycle(cycle, { status: ["current"], start_date: range, end_date: range }), true);
  assert.equal(shouldFilterCycle(cycle, { status: ["draft"] }), false);
  assert.equal(shouldFilterCycle({ ...cycle, start_date: null }, { start_date: range }), false);
});

void test("date ranges are inclusive, reject missing or invalid dates, and ignore empty selections", () => {
  assert.equal(matchesDateFilters("2026-10-01", range), true);
  assert.equal(matchesDateFilters("2026-10-31", range), true);
  assert.equal(matchesDateFilters("2026-09-30", range), false);
  assert.equal(matchesDateFilters("2026-11-01", range), false);
  assert.equal(matchesDateFilters(undefined, range), false);
  assert.equal(matchesDateFilters("invalid", range), false);
  assert.equal(matchesDateFilters(undefined, []), true);
  assert.equal(matchesDateFilters("2026-10-01", ["invalid;after"]), false);
});

void test("date shortcuts preserve relative thresholds and reject unsupported operators", (t) => {
  t.mock.timers.enable({ apis: ["Date"], now: new Date("2026-10-31T12:00:00Z") });
  assert.equal(satisfiesDateFilter(new Date(), "today;custom;custom"), true);
  assert.equal(satisfiesDateFilter(new Date("2026-10-30T12:00:00Z"), "yesterday;custom;custom"), true);
  assert.equal(satisfiesDateFilter(new Date("2026-10-24T12:00:00Z"), "last_7_days;custom;custom"), true);
  assert.equal(satisfiesDateFilter(new Date("2026-10-23T12:00:00Z"), "last_7_days;custom;custom"), false);
  assert.equal(satisfiesDateFilter(new Date("2026-10-24T12:00:00Z"), "1_weeks;before;fromnow"), true);
  assert.equal(satisfiesDateFilter(new Date("2026-11-07T12:00:00Z"), "1_weeks;after;fromnow"), true);
  assert.equal(satisfiesDateFilter(new Date("2026-10-24T12:00:00Z"), "unknown;before;fromnow"), false);
  assert.equal(satisfiesDateFilter(new Date("2026-08-01T12:00:00Z"), "2_months;before;fromnow"), false);
  assert.equal(satisfiesDateFilter(new Date(), "2026-10-01;unknown"), false);
});

void test("filter counts count selected values and enabled switches without interpreting other values", () => {
  assert.equal(
    calculateTotalFilters({
      members: ["one", "two"],
      enabled: true,
      disabled: false,
      empty: null,
      query: "text",
      zero: 0,
    }),
    3
  );
  assert.equal(calculateTotalFilters(undefined), 0);
  assert.equal(calculateTotalFilters({}), 0);
});

void test("cycle icons normalize known status codes and fall back safely for absent or unknown values", () => {
  for (const status of ["current", "upcoming", "completed", "draft"]) {
    assert.equal(getCycleGroup(status), status);
    assert.equal(getCycleGroup(status.toUpperCase()), status);
  }
  for (const status of [null, undefined, "", "future-status", 42]) assert.equal(getCycleGroup(status), "draft");
});

import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const services = new Map([
  ["@/services/cycle.service", "CycleService"],
  ["@/services/cycle_archive.service", "CycleArchiveService"],
  ["@/services/issue", "IssueService"],
  ["@/services/project", "ProjectService"],
]);
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (services.has(specifier)) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(`export class ${services.get(specifier)} {}`)}`,
      };
    }
    return nextResolve(specifier, context);
  },
});
const { CycleStore } = await import("../../apps/web/core/store/cycle.store.ts");

function archiveStore(response) {
  const store = new CycleStore({
    router: { projectId: "project" },
    cycleFilter: { archivedCyclesSearchQuery: "", getArchivedFiltersByProjectId: () => ({}) },
  });
  store.cycleArchiveService.getArchivedCycles = async () => response;
  return store;
}

test("opening archived cycles directly exposes fetched rows without loading the active list", async () => {
  const store = archiveStore([
    { id: "archive", project_id: "project", name: "Archived cycle", archived_at: "2026-01-08", sort_order: 1 },
  ]);
  assert.equal(store.currentProjectArchivedCycleIds, null);

  await store.fetchArchivedCycles("workspace", "project");

  assert.equal(store.loader, false);
  assert.deepEqual(store.currentProjectArchivedCycleIds, ["archive"]);
  assert.deepEqual(store.getFilteredArchivedCycleIds("project"), ["archive"]);
  assert.equal(store.currentProjectCycleIds, null, "an archive fetch does not load the active collection");
});

test("an empty archived response finishes loading independently for each project", async () => {
  const store = archiveStore([]);
  await store.fetchArchivedCycles("workspace", "project");
  assert.deepEqual(store.currentProjectArchivedCycleIds, []);
  assert.deepEqual(store.getFilteredArchivedCycleIds("project"), []);
  assert.equal(store.getFilteredArchivedCycleIds("other-project"), null);
});

test("a failed archive request remains retryable and is not treated as a loaded empty collection", async () => {
  const store = archiveStore([]);
  store.cycleArchiveService.getArchivedCycles = async () => {
    throw new Error("offline");
  };
  await store.fetchArchivedCycles("workspace", "project");
  assert.equal(store.loader, false);
  assert.equal(store.currentProjectArchivedCycleIds, null);
  store.cycleArchiveService.getArchivedCycles = async () => [];
  await store.fetchArchivedCycles("workspace", "project");
  assert.deepEqual(store.currentProjectArchivedCycleIds, []);
});

import assert from "node:assert/strict";
import test from "node:test";
import { act, identity, setup, settleQuery } from "./helpers/issue-detail-fixture.mjs";
for (const mode of ["detail", "peek"]) {
  for (const operation of ["addCycleToIssue", "removeIssueFromCycle"])
    void test(`${mode} ${operation} rejects transport failure without reporting success`, async (t) => {
      const view = await setup(t, mode);
      view.transport.assignment = async () => {
        throw new Error("assignment rejected");
      };
      await act(async () => {
        await assert.rejects(
          globalThis.detailWriteOperations[operation]("acme", "p1", "new-cycle", "item"),
          /assignment rejected/
        );
      });
      assert.equal(view.writes.length, 1);
      assert.equal(view.row().cycle_id, "old-cycle");
      assert.deepEqual(globalThis.detailWriteToasts, []);
    });
  for (const kind of ["cycle", "module"])
    void test(`${mode} ${kind} selection prevents repeat writes and recovers after failure`, async (t) => {
      const view = await setup(t, mode);
      const pending = Promise.withResolvers();
      view.transport.assignment = () => pending.promise;
      const control = view.property(kind);
      assert.ok(control);
      await act(() => {
        control.click();
        control.click();
      });
      await settleQuery();
      const count = view.writes.length;
      const disabled = control.disabled;
      await act(() => pending.reject(new Error("assignment rejected")));
      await settleQuery();
      assert.equal(count, 1);
      assert.equal(disabled, true);
      assert.equal(control.disabled, false);
      assert.deepEqual(globalThis.detailUnhandled, []);
      assert.equal(globalThis.detailWriteToasts.length, 1);
      assert.equal(globalThis.detailWriteToasts[0].type, "error");
      view.transport.assignment = async () => {};
      await view.click(view.property(kind));
      assert.equal(view.writes.length, 2);
      assert.deepEqual(
        JSON.parse(view.property(kind).getAttribute("data-value")),
        kind === "cycle" ? "new-cycle" : ["new-module"]
      );
    });
  void test(`${mode} assignments share pending state with ordinary property edits`, async (t) => {
    const view = await setup(t, mode);
    const pending = Promise.withResolvers();
    view.transport.assignment = () => pending.promise;
    await view.click(view.property("cycle"));
    const disabled = view.property("priority").disabled;
    await act(() => globalThis.detailWriteControls.get("priority").onChange("urgent"));
    await act(() => pending.resolve());
    await settleQuery();
    assert.equal(disabled, true);
    assert.equal(view.writes.length, 1);
    assert.equal(view.row().priority, "low");
  });
  void test(`${mode} late assignment errors do not notify a replacement session`, async (t) => {
    const view = await setup(t, mode);
    const pending = Promise.withResolvers();
    view.transport.assignment = () => pending.promise;
    await view.click(view.property("cycle"));
    await act(() => identity.advanceIdentitySession(view.client));
    await settleQuery();
    await act(() => pending.reject(new Error("assignment rejected")));
    await settleQuery();
    assert.deepEqual(globalThis.detailWriteToasts, []);
    assert.deepEqual(globalThis.detailUnhandled, []);
  });
  void test(`${mode} cycle removal reports one accepted result and reconciles its selection`, async (t) => {
    const view = await setup(t, mode);
    globalThis.assignmentCycleValue = null;
    await view.click(view.property("cycle"));
    assert.equal(view.writes.length, 1);
    assert.equal(view.row().cycle_id, null);
    assert.equal(view.property("cycle").getAttribute("data-value"), "null");
    assert.equal(globalThis.detailWriteToasts.length, 1);
    assert.equal(globalThis.detailWriteToasts[0].type, "success");
  });
  void test(`${mode} cannot send a module change through a different requested workspace`, async (t) => {
    const view = await setup(t, mode);
    await act(async () => {
      await assert.rejects(
        globalThis.detailWriteOperations.changeModulesInIssue("other", "p1", "item", ["new-module"], [])
      );
    });
    assert.deepEqual(view.writes, []);
  });
  void test(`${mode} captured assignment callbacks cannot edit an archived item`, async (t) => {
    const view = await setup(t, mode, { archived: true });
    await act(() => globalThis.detailWriteControls.get("cycle").onChange("new-cycle"));
    await settleQuery();
    assert.deepEqual(view.writes, []);
    assert.deepEqual(globalThis.detailUnhandled, []);
  });
  void test(`${mode} module changes preserve retained modules and deduplicate additions`, async (t) => {
    const view = await setup(t, mode);
    globalThis.assignmentModuleValue = ["old-module", "new-module", "new-module"];
    await view.click(view.property("module"));
    assert.deepEqual(view.writes, [
      ["module-change", "acme", "p1", "item", { modules: ["new-module"], removed_modules: [] }],
    ]);
    assert.deepEqual(view.row().module_ids, ["old-module", "new-module"]);
  });
  void test(`${mode} accepted assignments remain successful if a detail refresh fails`, async (t) => {
    const view = await setup(t, mode);
    view.transport.read = async () => {
      throw new Error("read offline");
    };
    await act(async () => {
      await globalThis.detailWriteOperations.addCycleToIssue("acme", "p1", "new-cycle", "item");
    });
    await settleQuery();
    assert.equal(view.writes.length, 1);
    assert.equal(view.row().cycle_id, "new-cycle");
    assert.deepEqual(globalThis.detailWriteToasts, []);
  });
}

for (const mode of ["detail", "peek"])
  void test(`${mode} a failed module write releases the control and reports one error`, async (t) => {
    const view = await setup(t, mode);
    view.transport.assignment = async () => {
      throw new Error("assignment rejected");
    };
    await view.click(view.property("module"));
    assert.equal(view.property("module").disabled, false);
    assert.equal(view.writes.length, 1);
    assert.deepEqual(globalThis.detailUnhandled, []);
    assert.equal(globalThis.detailWriteToasts.length, 1);
    assert.equal(globalThis.detailWriteToasts[0].type, "error");
  });

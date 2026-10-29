import assert from "node:assert/strict";
import test from "node:test";
import { act, identity, setup, settleQuery } from "./helpers/issue-detail-fixture.mjs";
async function rejectsWrite(t, mode) {
  const view = await setup(t, mode);
  view.transport.patch = async () => {
    throw new Error("write rejected");
  };
  await act(async () => {
    await assert.rejects(
      globalThis.detailWriteOperations.update("acme", "p1", "item", { priority: "urgent" }),
      /write rejected/
    );
  });
  assert.equal(view.writes.length, 1);
  assert.equal(view.row().priority, "low");
  assert.deepEqual(globalThis.detailWriteToasts, []);
}
await test("detail writes reject failures so the submitting form can retain its draft", (t) =>
  rejectsWrite(t, "detail"));
await test("preview writes reject failures so the submitting form can retain its draft", (t) =>
  rejectsWrite(t, "peek"));
async function rejectsParentSelection(t, mode) {
  const view = await setup(t, mode);
  view.transport.patch = async () => {
    throw new Error("write rejected");
  };
  await act(async () => {
    await assert.rejects(
      globalThis.detailWriteSelection.onSelect({ id: "parent", project_id: "p2" }),
      /write rejected/
    );
  });
  assert.equal(view.row().parent_id, null);
  assert.equal(view.writes.length, 1);
  assert.deepEqual(globalThis.detailWriteToasts, []);
}
await test("detail parent selection receives the rejected write", (t) => rejectsParentSelection(t, "detail"));
await test("preview parent selection receives the rejected write", (t) => rejectsParentSelection(t, "peek"));
async function pendingProperty(t, mode) {
  const view = await setup(t, mode);
  const pending = Promise.withResolvers();
  view.transport.patch = () => pending.promise;
  const button = view.property("priority");
  await act(async () => {
    button.click();
    button.click();
    await Promise.resolve();
  });
  await settleQuery();
  const writes = view.writes.length;
  const disabled = view.property("priority").disabled;
  await act(() => pending.reject(new Error("write rejected")));
  await settleQuery();
  assert.equal(writes, 1);
  assert.equal(disabled, true);
  assert.equal(view.property("priority").disabled, false);
  assert.equal(globalThis.detailWriteToasts.length, 1);
  assert.equal(globalThis.detailWriteToasts[0].type, "error");
  view.transport.patch = async () => {};
  await view.click(view.property("priority"));
  assert.equal(view.writes.length, 2);
  assert.equal(view.property("priority").getAttribute("data-value"), '"urgent"');
}
await test("detail property edits prevent duplicate writes and recover after failure", (t) =>
  pendingProperty(t, "detail"));
await test("preview property edits prevent duplicate writes and recover after failure", (t) =>
  pendingProperty(t, "peek"));
async function lateFailure(t, mode) {
  const view = await setup(t, mode);
  const pending = Promise.withResolvers();
  view.transport.patch = () => pending.promise;
  await view.click(view.property("estimate"));
  await act(() => identity.advanceIdentitySession(view.client));
  await settleQuery();
  await act(() => pending.reject(new Error("old failure")));
  await settleQuery();
  assert.equal(view.writes.length, 1);
  assert.deepEqual(globalThis.detailWriteToasts, []);
}
await test("late detail estimate failures cannot notify a replacement session", (t) => lateFailure(t, "detail"));
await test("late preview estimate failures cannot notify a replacement session", (t) => lateFailure(t, "peek"));
await test("a detail write cannot silently target the route workspace instead of its requested workspace", async (t) => {
  const view = await setup(t);
  await act(async () => {
    await assert.rejects(
      globalThis.detailWriteOperations.update("other-workspace", "p1", "item", { priority: "urgent" })
    );
  });
  assert.deepEqual(view.writes, []);
});

for (const mode of ["detail", "peek"]) {
  void test(`${mode} inline state, assignee, date and estimate failures stay handled`, async (t) => {
    const view = await setup(t, mode);
    view.transport.patch = async () => {
      throw new Error("write rejected");
    };
    await view.click(view.property("state"));
    await view.click(view.property("assignee"));
    await view.click(view.property("date"));
    await view.click(view.property("estimate"));
    assert.equal(view.writes.length, 4);
    assert.equal(globalThis.detailWriteToasts.length, 4);
    assert.deepEqual(view.row().assignee_ids, []);
    assert.equal(view.row().start_date, null);
    assert.equal(view.row().estimate_point, "small");
  });
  void test(`${mode} estimate edits share the pending guard with basic properties`, async (t) => {
    const view = await setup(t, mode);
    const pending = Promise.withResolvers();
    view.transport.patch = () => pending.promise;
    const estimate = globalThis.detailWriteControls.get("estimate").onChange;
    await view.click(view.property("priority"));
    const estimateDisabled = view.property("estimate").disabled;
    await act(() => estimate("large"));
    await act(() => pending.resolve());
    await settleQuery();
    assert.equal(estimateDisabled, true);
    assert.equal(view.writes.length, 1);
    assert.equal(view.row().priority, "urgent");
    assert.equal(view.row().estimate_point, "small");
    assert.deepEqual(globalThis.detailWriteToasts, []);
  });
  void test(`${mode} captured writes reject after the account session changes`, async (t) => {
    const view = await setup(t, mode);
    const update = globalThis.detailWriteOperations.update;
    await act(() => identity.advanceIdentitySession(view.client));
    await settleQuery();
    await assert.rejects(update("acme", "p1", "item", { priority: "urgent" }));
    assert.deepEqual(view.writes, []);
  });
  void test(`${mode} archived items reject direct edits and disable property controls`, async (t) => {
    const view = await setup(t, mode, { archived: true });
    assert.equal(view.property("priority").disabled, true);
    await assert.rejects(globalThis.detailWriteOperations.update("acme", "p1", "item", { priority: "urgent" }));
    assert.deepEqual(view.writes, []);
  });
  void test(`${mode} parent-summary unlink handles rejection and prevents repeat submissions`, async (t) => {
    const view = await setup(t, mode, { parent: true });
    const pending = Promise.withResolvers();
    view.transport.patch = () => pending.promise;
    const button = [...view.environment.container.querySelectorAll("button")].find(
      (b) => b.textContent === "issue.remove.parent.label"
    );
    assert.ok(button);
    await act(() => {
      button.click();
      button.click();
    });
    await settleQuery();
    const writes = view.writes.length;
    const disabled = button.disabled;
    await act(() => pending.reject(new Error("write rejected")));
    await settleQuery();
    assert.equal(writes, 1);
    assert.equal(disabled, true);
    assert.equal(button.disabled, false);
    assert.equal(view.row().parent_id, "parent");
    assert.equal(globalThis.detailWriteToasts.length, 1);
  });
  void test(`${mode} readonly parent-summary unlink cannot write`, async (t) => {
    const view = await setup(t, mode, { parent: true, role: 5 });
    const button = [...view.environment.container.querySelectorAll("button")].find(
      (b) => b.textContent === "issue.remove.parent.label"
    );
    assert.ok(button);
    assert.equal(button.disabled, true);
    await view.click(button);
    assert.deepEqual(view.writes, []);
  });
}

for (const mode of ["detail", "peek"])
  void test(`${mode} a late accepted write cannot invalidate a replacement session's schema`, async (t) => {
    const view = await setup(t, mode);
    const pending = Promise.withResolvers();
    view.transport.patch = () => pending.promise;
    let write;
    await act(() => {
      write = globalThis.detailWriteOperations.update("acme", "p1", "item", { priority: "urgent" });
    });
    await act(() => identity.advanceIdentitySession(view.client));
    await settleQuery();
    const key = ["workspace", "acme", "work-item-schema", "replacement", { issue_id: "item" }, "detail"];
    view.client.setQueryData(key, { properties: [] });
    await act(async () => {
      pending.resolve();
      await write;
    });
    await settleQuery();
    assert.equal(view.client.getQueryState(key).isInvalidated, false);
    assert.deepEqual(globalThis.detailWriteToasts, []);
  });

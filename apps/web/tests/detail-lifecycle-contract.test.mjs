import assert from "node:assert/strict";
import test from "node:test";
import { setup, React, act, identity, settleQuery } from "./helpers/issue-lifecycle-fixture.mjs";
for (const mode of ["detail", "peek"]) {
  void test(`${mode} delete rejection stays in the actual confirmation dialog with one error`, async (t) => {
    const view = await setup(t, mode);
    view.transport.write = async () => {
      throw new Error("write rejected");
    };
    await view.click(view.action("delete"));
    assert.ok(view.confirm());
    await view.click(view.confirm());
    assert.equal(view.state.writes.length, 1);
    assert.equal(Boolean(view.dialog()), true);
    assert.equal(view.confirm()?.disabled, false);
    assert.equal(view.state.toasts.length, 1);
    assert.equal(view.state.toasts[0].type, "error");
    assert.equal(view.state.closed, 0);
  });
  void test(`${mode} restore rejection reports one error without success or dismissal`, async (t) => {
    const view = await setup(t, mode, { archived: true });
    view.transport.write = async () => {
      throw new Error("write rejected");
    };
    await view.click(view.action("restore"));
    assert.equal(view.state.writes.length, 1);
    assert.equal(view.state.toasts.length, 1);
    assert.equal(view.state.toasts[0].type, "error");
    assert.equal(view.state.closed, 0);
    assert.deepEqual(view.state.unhandled, []);
  });
  void test(`${mode} restore prevents duplicate writes and becomes retryable after failure`, async (t) => {
    const view = await setup(t, mode, { archived: true });
    const pending = Promise.withResolvers();
    void pending.promise.catch(() => {});
    view.transport.write = () => pending.promise;
    const action = view.action("restore");
    await act(() => {
      action.click();
      action.click();
    });
    await settleQuery();
    const writes = view.state.writes.length;
    const disabled = view.action("restore").disabled;
    await act(() => pending.reject(new Error("write rejected")));
    await settleQuery();
    assert.equal(writes, 1);
    assert.equal(disabled, true);
    assert.equal(view.action("restore").disabled, false);
    assert.deepEqual(view.state.unhandled, []);
  });
  void test(`${mode} a late restore failure cannot notify a replacement session`, async (t) => {
    const view = await setup(t, mode, { archived: true });
    const pending = Promise.withResolvers();
    void pending.promise.catch(() => {});
    view.transport.write = () => pending.promise;
    await view.click(view.action("restore"));
    await act(() => identity.advanceIdentitySession(view.client));
    await settleQuery();
    await act(() => pending.reject(new Error("write rejected")));
    await settleQuery();
    assert.deepEqual(view.state.toasts, []);
    assert.equal(view.state.closed, 0);
  });
}

for (const mode of ["detail", "peek"])
  void test(`archived ${mode} restoration does not require an archive-list preferences provider`, async (t) => {
    const view = await setup(t, mode, { archived: true, noArchivePrefs: true });
    await view.click(view.action("restore"));
    assert.equal(view.state.writes.length, 1);
    assert.equal(view.state.writes[0][0], "restore");
    assert.equal(view.state.toasts.filter((toast) => toast.type === "error").length, 0);
  });
for (const mode of ["detail", "peek"]) {
  void test(`${mode} archive failure leaves its confirmation open for retry`, async (t) => {
    const view = await setup(t, mode);
    view.transport.write = async () => {
      throw new Error("archive rejected");
    };
    await view.click(view.action("archive"));
    const confirm = [...view.dialog().querySelectorAll("button")].find((b) => b.textContent === "common.archive");
    await view.click(confirm);
    assert.equal(view.state.writes.length, 1);
    assert.equal(Boolean(view.dialog()), true);
    assert.equal(confirm.disabled, false);
    assert.equal(view.state.toasts.length, 1);
    assert.equal(view.state.toasts[0].type, "error");
    assert.equal(view.state.closed, 0);
  });
  void test(`${mode} accepted restore survives a failed ancillary read`, async (t) => {
    const view = await setup(t, mode, { archived: true });
    view.transport.read = async () => {
      throw new Error("read unavailable");
    };
    await view.click(view.action("restore"));
    assert.equal(view.state.writes.length, 1);
    assert.equal(
      view.state.toasts.some((toast) => toast.type === "error"),
      false
    );
    assert.deepEqual(view.state.unhandled, []);
  });
}
for (const kind of ["archive", "restore"]) {
  void test(`a late peek ${kind} cannot close the same item in a replacement project`, async (t) => {
    const view = await setup(t, "peek", { archived: kind === "restore" });
    const pending = Promise.withResolvers();
    view.transport.write = () => pending.promise;
    await view.click(view.action(kind));
    if (kind === "archive")
      await view.click([...view.dialog().querySelectorAll("button")].find((b) => b.textContent === "common.archive"));
    assert.equal(view.state.writes.length, 1);
    view.state.peek = { ...view.state.peek, projectId: "p2" };
    await view.render();
    await settleQuery();
    await act(() => pending.resolve());
    await settleQuery();
    assert.equal(view.state.closed, 0);
    assert.equal(view.state.peek.projectId, "p2");
  });
  void test(`a late peek ${kind} cannot close an item after a selection round trip`, async (t) => {
    const view = await setup(t, "peek", { archived: kind === "restore" });
    const pending = Promise.withResolvers();
    view.transport.write = () => pending.promise;
    await view.click(view.action(kind));
    if (kind === "archive")
      await view.click([...view.dialog().querySelectorAll("button")].find((b) => b.textContent === "common.archive"));
    assert.equal(view.state.writes.length, 1);
    view.state.peek = { ...view.state.peek, issueId: "replacement" };
    await view.render();
    view.state.peek = { ...view.state.peek, issueId: "item" };
    await view.render();
    await act(() => pending.resolve());
    await settleQuery();
    assert.equal(view.state.closed, 0);
    assert.equal(view.state.peek.issueId, "item");
  });
}
for (const kind of ["delete", "archive", "restore"])
  void test(`a late detail ${kind} cannot redirect a replacement route`, async (t) => {
    const view = await setup(t, "detail", { archived: kind === "restore" });
    const pending = Promise.withResolvers();
    view.transport.write = () => pending.promise;
    await view.click(view.action(kind));
    if (kind === "delete") await view.click(view.confirm());
    if (kind === "archive")
      await view.click([...view.dialog().querySelectorAll("button")].find((b) => b.textContent === "common.archive"));
    assert.equal(view.state.writes.length, 1);
    await act(() => view.router.navigate({ to: "/acme/projects/p1/issues/P1-2" }));
    await settleQuery();
    await act(() => pending.resolve());
    await settleQuery();
    assert.equal(view.router.state.location.pathname, "/acme/projects/p1/issues/P1-2");
  });
for (const parent of ["cycle", "module"])
  for (const kind of ["delete", "archive"])
    void test(`peek ${kind} retains captured ${parent} statistics refresh`, async (t) => {
      const view = await setup(t, "peek", { parent });
      const reads = view.state.parentReads;
      assert.ok(reads > 0);
      await view.click(view.action(kind));
      if (kind === "delete") await view.click(view.confirm());
      else
        await view.click([...view.dialog().querySelectorAll("button")].find((b) => b.textContent === "common.archive"));
      assert.equal(view.state.writes.length, 1);
      assert.ok(view.state.parentReads > reads);
      assert.equal(
        view.state.toasts.some((toast) => toast.type === "error"),
        false
      );
    });
void test("accepted deletion is not retried when opening its list fails", async (t) => {
  const view = await setup(t, "detail");
  t.mock.method(view.router, "navigate", async () => {
    throw new Error("navigation failed");
  });
  await view.click(view.action("delete"));
  await view.click(view.confirm());
  assert.equal(view.state.writes.length, 1);
  assert.equal(Boolean(view.dialog()), false);
  assert.match(
    view.state.toasts.find((toast) => toast.type === "error")?.message ?? "",
    /deleted.*list could not be opened/
  );
  assert.deepEqual(view.state.unhandled, []);
});
void test("workspace draft delete failure keeps its native confirmation retryable", async (t) => {
  const view = await setup(t, "detail");
  const { WorkspaceDraftIssueDeleteIssueModal } =
    await import("../core/components/issues/workspace-draft/delete-modal.tsx");
  await view.renderElement(
    React.createElement(WorkspaceDraftIssueDeleteIssueModal, {
      isOpen: true,
      data: { id: "draft", created_by: "alice" },
      handleClose: () => view.state.closed++,
      onSubmit: async () => {
        throw new Error("delete rejected");
      },
    })
  );
  await view.click(view.confirm());
  assert.equal(Boolean(view.dialog()), true);
  assert.equal(view.confirm().disabled, false);
  assert.equal(view.state.closed, 0);
  assert.equal(view.state.toasts.length, 1);
  assert.equal(view.state.toasts[0].type, "error");
});
void test("peek text edits reconcile the canonical accepted response", async (t) => {
  const view = await setup(t, "peek");
  view.transport.canonical = { name: "Canonical title" };
  await act(() => globalThis.lifecycleOperations.updateText("acme", "p1", "item", { name: "Requested title" }));
  await settleQuery();
  const query = view.client
    .getQueryCache()
    .findAll({ queryKey: ["issue-details"] })
    .find((q) => q.queryKey.at(-2) === "detail" && q.queryKey.at(-1) === "item");
  assert.equal(query.state.data.name, "Canonical title");
});
void test("read-only archived actions do not expose restoration", async (t) => {
  const view = await setup(t, "detail", { archived: true });
  const { ArchivedIssueQuickActions } =
    await import("../core/components/issues/issue-layouts/quick-action-dropdowns/archived-issue.tsx");
  await view.renderElement(
    React.createElement(ArchivedIssueQuickActions, {
      issue: { id: "item", project_id: "p1", archived_at: "2026-10-01" },
      readOnly: true,
      handleRestore: async () => {
        view.state.writes.push("unexpected");
      },
    })
  );
  assert.equal(Boolean(view.action("restore")), false);
  assert.deepEqual(view.state.writes, []);
});
void test("a missing restore action cannot report success", async (t) => {
  const view = await setup(t, "detail", { archived: true });
  const { WorkItemDetailQuickActions } =
    await import("../core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx");
  await view.renderElement(
    React.createElement(WorkItemDetailQuickActions, {
      issue: { id: "item", project_id: "p1", archived_at: "2026-10-01" },
    })
  );
  assert.equal(view.action("restore").disabled, true);
  await view.click(view.action("restore"));
  assert.deepEqual(view.state.toasts, []);
});
void test("an accepted peek deletion stays successful if notification dismissal fails", async (t) => {
  const errors = t.mock.method(console, "error", () => {});
  const view = await setup(t, "peek", {
    onDismiss: () => {
      throw new Error("dismiss failed");
    },
  });
  await view.click(view.action("delete"));
  await view.click(view.confirm());
  assert.equal(view.state.writes.length, 1);
  assert.equal(view.state.closed, 1);
  assert.equal(
    view.state.toasts.some((toast) => toast.type === "error"),
    false
  );
  assert.deepEqual(view.state.unhandled, []);
  assert.equal(errors.mock.callCount(), 1);
});

for (const mode of ["detail", "peek"])
  void test(`archived ${mode} deletion does not require archive-list preferences`, async (t) => {
    const view = await setup(t, mode, { archived: true, noArchivePrefs: true });
    await view.click(view.action("delete"));
    await view.click(view.confirm());
    assert.equal(view.state.writes.length, 1);
    assert.equal(view.state.writes[0][0], "delete");
    assert.equal(
      view.state.toasts.some((toast) => toast.type === "error"),
      false
    );
    assert.equal(Boolean(view.dialog()), false);
  });

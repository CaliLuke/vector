import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  settleQuery,
  fixture,
  button,
  click,
  changeName,
  deferred,
  scope,
  alpha,
  beta,
  labelKeys,
  IssueLabelService,
} from "./helpers/labels-native-ui-fixture.mjs";

await test("failed native label loading exposes Retry and recovers through the real settings consumer", async (t) => {
  let offline = true;
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => {
    if (offline) throw new Error("offline");
    return [];
  });
  const view = await fixture(t);
  await view.render(React.createElement(view.ProjectSettingsLabelList));
  assert.ok(view.environment.container.querySelector('[role="alert"]'));
  assert.equal(view.environment.container.querySelector('[data-testid="label-loader"]'), null);
  offline = false;
  await click(button(view.environment.container, "Retry"));
  assert.match(view.environment.container.textContent, /No labels/);
  assert.equal(view.environment.container.querySelector('[role="alert"]'), null);
  assert.deepEqual(view.client.getQueryData(labelKeys.project(scope)), []);
});

await test("accepted deletion of A cannot close a reopened B dialog", async (t) => {
  const pending = deferred();
  const started = deferred();
  t.mock.method(IssueLabelService.prototype, "deleteIssueLabel", async (_workspace, _project, id) => {
    started.resolve(id);
    await pending.promise;
  });
  const view = await fixture(t, [alpha, beta]);
  await view.render(React.createElement(view.ProjectSettingsLabelList));
  await click(button(view.environment.container, "Delete Alpha"));
  await click(button(view.environment.container, "Delete"));
  assert.equal(await started.promise, alpha.id);
  await click(button(view.environment.container, "Cancel"));
  await click(button(view.environment.container, "Delete Beta"));
  assert.match(view.environment.container.querySelector('[role="dialog"]').textContent, /Beta/);
  assert.equal(
    (button(view.environment.container, "Delete") ?? button(view.environment.container, "Deleting"))?.disabled,
    false,
    "B owns a fresh enabled delete action while A is pending"
  );
  await act(async () => pending.resolve());
  await settleQuery();
  assert.match(view.environment.container.querySelector('[role="dialog"]')?.textContent ?? "", /Beta/);
  assert.equal(button(view.environment.container, "Delete")?.disabled, false);
  assert.deepEqual(
    view.client.getQueryData(labelKeys.project(scope)).map((label) => label.id),
    [beta.id]
  );
});

function holdCreation(t) {
  const pending = deferred();
  const started = deferred();
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => []);
  t.mock.method(IssueLabelService.prototype, "createIssueLabel", async (_workspace, projectId, data) => {
    started.resolve({ projectId, data });
    await pending.promise;
    return { ...alpha, name: data.name };
  });
  return { pending, started };
}
async function submitName(view, name) {
  await changeName(view.environment.container, name);
  await act(async () =>
    view.environment.container
      .querySelector("form")
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
}

await test("an inline new-label draft resets on project change and accepted old work cannot close its replacement", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(view.ProjectSettingsLabelList));
  await click(button(view.environment.container, "common.add_label"));
  await changeName(view.environment.container, "Draft A");
  await act(async () =>
    view.environment.container
      .querySelector("form")
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  assert.equal((await started.promise).projectId, "p1");
  await act(async () => view.router.navigate({ to: "/acme/settings/projects/p2/labels" }));
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value ?? "", "");
  if (!view.environment.container.querySelector("form"))
    await click(button(view.environment.container, "common.add_label"));
  await changeName(view.environment.container, "Draft B");
  assert.equal(view.environment.container.querySelector('input[name="name"]').disabled, false);
  await act(async () => pending.resolve());
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value, "Draft B");
  assert.equal(view.client.getQueryData(labelKeys.project(scope))[0].name, "Draft A");
  assert.deepEqual(view.client.getQueryData(labelKeys.project({ ...scope, projectId: "p2" })), []);
});

await test("same-project inline replacement survives an accepted submission from its closed predecessor", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(React.StrictMode, null, React.createElement(view.ProjectSettingsLabelList)));
  await click(button(view.environment.container, "common.add_label"));
  await submitName(view, "Draft A");
  assert.equal((await started.promise).data.name, "Draft A");
  await click(button(view.environment.container, "common.add_label"));
  assert.equal(view.environment.container.querySelector('input[name="name"]').value, "");
  await changeName(view.environment.container, "Draft B");
  assert.equal(view.environment.container.querySelector('input[name="name"]').disabled, false);
  await act(async () => pending.resolve());
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value, "Draft B");
  assert.equal(view.client.getQueryData(labelKeys.project(scope))[0].name, "Draft A");
});

await test("returning to the original project does not revive an earlier inline completion", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(view.ProjectSettingsLabelList));
  await click(button(view.environment.container, "common.add_label"));
  await submitName(view, "Draft A");
  assert.equal((await started.promise).projectId, "p1");
  await act(async () => view.router.navigate({ to: "/acme/settings/projects/p2/labels" }));
  await settleQuery();
  await act(async () => view.router.navigate({ to: "/acme/settings/projects/p1/labels" }));
  await settleQuery();
  await click(button(view.environment.container, "common.add_label"));
  await changeName(view.environment.container, "Draft B");
  assert.equal(view.environment.container.querySelector('input[name="name"]').disabled, false);
  await act(async () => pending.resolve());
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value, "Draft B");
  assert.equal(view.client.getQueryData(labelKeys.project(scope))[0].name, "Draft A");
});

await test("rejected work from an unmounted inline draft cannot toast into its replacement", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(React.StrictMode, null, React.createElement(view.ProjectSettingsLabelList)));
  await click(button(view.environment.container, "common.add_label"));
  await submitName(view, "Draft A");
  assert.equal((await started.promise).data.name, "Draft A");
  await click(button(view.environment.container, "common.add_label"));
  await changeName(view.environment.container, "Draft B");
  await act(async () => pending.reject(new Error("late failure")));
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value, "Draft B");
  assert.deepEqual(globalThis.nativeLabelsUiToasts, []);
  assert.deepEqual(view.client.getQueryData(labelKeys.project(scope)), []);
});

await test("a failed current inline draft remains editable and exposes its failure after effect replay", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(React.StrictMode, null, React.createElement(view.ProjectSettingsLabelList)));
  await click(button(view.environment.container, "common.add_label"));
  await submitName(view, "Draft A");
  assert.equal((await started.promise).data.name, "Draft A");
  await act(async () => pending.reject(new Error("save failed")));
  await settleQuery();
  const input = view.environment.container.querySelector('input[name="name"]');
  assert.equal(input?.value, "Draft A");
  assert.equal(input.disabled, false);
  assert.equal(globalThis.nativeLabelsUiToasts.length, 1);
  assert.equal(globalThis.nativeLabelsUiToasts[0].type, "error");
  assert.deepEqual(view.client.getQueryData(labelKeys.project(scope)), []);
});

for (const unmountBeforeFailure of [false, true]) {
  void test(`remove-from-group feedback belongs to its ${unmountBeforeFailure ? "unmounted" : "current"} label item`, async (t) => {
    const view = await fixture(t, [alpha]);
    const { ProjectSettingLabelItem } =
      await import("../../apps/web/core/components/labels/project-setting-label-item.tsx");
    const pending = deferred();
    const updateLabel = t.mock.fn(() => pending.promise);
    t.mock.method(IssueLabelService.prototype, "patchIssueLabel", () => pending.promise);
    const props = {
      label: { ...alpha, parent: "parent-label" },
      isChild: true,
      isLastChild: true,
      isEditable: true,
      handleLabelDelete() {},
      setIsUpdating() {},
      onDrop() {},
      labelOperationsCallbacks: { createLabel: async () => alpha, updateLabel },
    };
    await view.render(React.createElement(React.StrictMode, null, React.createElement(ProjectSettingLabelItem, props)));
    await click(button(view.environment.container, "Remove from group"));
    assert.deepEqual(
      updateLabel.mock.calls.map((call) => call.arguments),
      [[alpha.id, { parent: null }]],
      "the settings owner receives the update"
    );
    if (unmountBeforeFailure) await view.render(React.createElement("p", null, "Replacement settings"));
    await act(async () => pending.reject(new Error("save failed")));
    await settleQuery();
    assert.equal(globalThis.nativeLabelsUiToasts.length, unmountBeforeFailure ? 0 : 1);
    if (!unmountBeforeFailure) assert.equal(globalThis.nativeLabelsUiToasts[0].type, "error");
  });
}

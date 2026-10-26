import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl as change, submitForm as submit } from "./helpers/native-forms.mjs";
import { setup, issueProps } from "./helpers/issue-form-harness.mjs";
import { enableTypeFields, propertyDefinition, customControl } from "./helpers/issue-custom-fields.mjs";

await test("saved source refresh cannot upgrade the version underneath local edits", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("effort", "Number")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const calls = [];
  const render = (version, name, effort) =>
    env.render(
      React.createElement(
        IssueFormRoot,
        issueProps({
          sourceVersion: version,
          data: {
            id: "existing",
            project_id: "first",
            type_id: "task",
            name,
            description_html: "<p>Original</p>",
            custom_properties: { effort },
          },
          onSubmit: async (...args) => calls.push(args),
        })
      )
    );
  await render("v1", "Saved title", 1);
  await change(env.container.querySelector('[name="name"]'), "Local title");
  await change(customControl(env.container, "effort"), "2");
  await render("v2", "Concurrent title", 9);
  assert.equal(env.container.querySelector('[name="name"]').value, "Local title");
  assert.equal(customControl(env.container, "effort").value, "2");
  await submit(env.container);
  assert.equal(calls[0][5], "v1");
  assert.equal(calls[0][0].name, "Local title");
  assert.deepEqual(calls[0][0].custom_properties, { effort: 2 });
});

await test("source conflict retains edits until an explicit successful reload", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("effort", "Number")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const calls = [];
  let failures = true;
  function Harness() {
    const [version, setVersion] = React.useState("v1");
    return React.createElement(
      IssueFormRoot,
      issueProps({
        key: version,
        sourceVersion: version,
        data: {
          id: "existing",
          project_id: "first",
          type_id: "task",
          name: version === "v1" ? "Saved" : "Concurrent",
          description_html: "<p></p>",
          custom_properties: { effort: version === "v1" ? 1 : 9 },
        },
        onSubmit: async (...args) => {
          calls.push(args);
          if (args[5] === "v1")
            throw { response: { status: 409, data: { fields: { source_changed: true, error: "Reload required." } } } };
        },
        onReloadSource: async () => {
          if (failures) throw new Error("Offline");
          setVersion("v2");
        },
      })
    );
  }
  await env.render(React.createElement(Harness));
  await change(env.container.querySelector('[name="name"]'), "Local");
  await change(customControl(env.container, "effort"), "2");
  await submit(env.container);
  const reload = () =>
    [...env.container.querySelectorAll("button")].find((b) => b.textContent.includes("Discard changes and reload"));
  assert.ok(reload());
  await act(async () => reload().click());
  assert.equal(env.container.querySelector('[name="name"]').value, "Local");
  assert.equal(customControl(env.container, "effort").value, "2");
  failures = false;
  await act(async () => reload().click());
  assert.equal(env.container.querySelector('[name="name"]').value, "Concurrent");
  assert.equal(customControl(env.container, "effort").value, "9");
  await change(env.container.querySelector('[name="name"]'), "New edit");
  await submit(env.container);
  assert.equal(calls.at(-1)[5], "v2");
});

async function sourceHarness(t, sourceOverrides = {}) {
  const env = await setup(t);
  const { useIssueModalSource } =
    await import("../../apps/web/core/components/issues/issue-modal/use-issue-modal-source.ts");
  const { workItemEditsService } = await import("../../apps/web/core/services/work-item-edits.service.ts");
  let source;
  function Harness() {
    const current = useIssueModalSource({
      data: { id: "existing", project_id: "first" },
      isOpen: true,
      fetchIssueDetails: false,
      workspaceSlug: "workspace",
      allowedProjectIds: ["first"],
      getProjectByIdentifier: () => undefined,
      fetchIssue: async () => undefined,
      ...sourceOverrides,
    });
    React.useLayoutEffect(() => {
      source = current;
    });
    return React.createElement(
      "output",
      null,
      JSON.stringify({ fields: current.fields, version: current.sourceVersion, access: current.accessState })
    );
  }
  return {
    ...env,
    service: workItemEditsService,
    source: () => source,
    start: () => env.render(React.createElement(Harness)),
    result: () => JSON.parse(env.container.querySelector("output").textContent),
  };
}
await test("transient source reload keeps canonical defaults, but denied refresh hides them", async (t) => {
  const f = await sourceHarness(t);
  const saved = { id: "existing", name: "Private", description_html: "<p>Saved</p>", custom_properties: { effort: 0 } };
  let failure;
  t.mock.method(f.service, "source", async () => {
    if (failure) throw failure;
    return { item: saved, source_version: "v1" };
  });
  await f.start();
  assert.deepEqual(f.result().fields, saved);
  failure = { response: { status: 500 } };
  await act(async () => {
    await assert.rejects(f.source().descriptionQuery.refetch({ throwOnError: true }));
  });
  assert.deepEqual(f.result().fields, saved);
  assert.equal(f.result().version, "v1");
  failure = { response: { status: 403 } };
  await act(async () => {
    await assert.rejects(f.source().descriptionQuery.refetch({ throwOnError: true }));
  });
  assert.equal(f.result().fields, undefined);
  assert.equal(f.result().version, undefined);
});
await test("late source response cannot restore data after workspace membership revocation", async (t) => {
  const f = await sourceHarness(t),
    response = Promise.withResolvers();
  const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
  t.mock.method(f.service, "source", () => response.promise);
  await f.start();
  await act(async () => {
    f.client.setQueryData(permissionKeys.workspaceMembership("workspace-query-test-user", "workspace"), {
      is_active: false,
      role: 20,
      updated_at: "v2",
    });
    response.resolve({ item: { id: "existing", name: "Private" }, source_version: "v1" });
  });
  assert.equal(f.result().fields, undefined);
  assert.equal(f.result().access, "denied");
});
await test("copying a published item does not overwrite creation identity or title", async (t) => {
  const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
  t.mock.method(IssueService.prototype, "retrieve", async () => ({
    id: "original",
    project_id: "first",
    name: "Original",
    description_html: "<p>Copy body</p>",
    type_id: "task",
    custom_properties: { effort: 0 },
  }));
  const f = await sourceHarness(t, {
    data: { sourceIssueId: "original", project_id: "first", name: "Copy of original" },
    fetchIssueDetails: true,
  });
  t.mock.method(f.service, "source", async () => {
    throw new Error("Copy used saved editor source");
  });
  await f.start();
  await act(async () => f.source().descriptionQuery.refetch({ throwOnError: true }));
  assert.deepEqual(f.result().fields, {
    description_html: "<p>Copy body</p>",
    type_id: "task",
    custom_properties: { effort: 0 },
  });
  assert.equal(f.result().version, undefined);
});

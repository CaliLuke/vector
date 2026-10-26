import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl as change, submitForm as submit } from "./helpers/native-forms.mjs";
import { setup, state, issueProps, formObserver, formSources } from "./helpers/issue-form-harness.mjs";

import {
  propertyDefinition,
  enableTypeFields,
  customControl,
  chooseWorkItemType,
} from "./helpers/issue-custom-fields.mjs";

await test("custom values retain decimal input, zero, false and inactive type drafts until submission", async (t) => {
  const env = await setup(t);
  enableTypeFields((input) => [
    propertyDefinition(input.type_id === "bug" ? "severity" : "effort", "Number", { default_value: 4 }),
    propertyDefinition("verified", "Boolean"),
  ]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  let saved;
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        data: { name: "New", description_html: "<p></p>" },
        onSubmit: async (value) => {
          saved = value;
        },
      })
    )
  );
  await change(customControl(env.container, "effort"), "1.");
  assert.equal(customControl(env.container, "effort").value, "1.");
  await change(customControl(env.container, "effort"), "0");
  await act(async () => customControl(env.container, "verified").click());
  await act(async () => customControl(env.container, "verified").click());
  await chooseWorkItemType(env.container, "bug");
  await change(customControl(env.container, "severity"), "7");
  await chooseWorkItemType(env.container, "task");
  assert.equal(customControl(env.container, "effort").value, "0");
  await submit(env.container);
  assert.deepEqual(saved.custom_properties, { effort: 0, verified: false });
});

await test("custom-field save errors retain input and focus the field after unlocking", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("effort", "Number"), propertyDefinition("cost", "Number")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const submissions = [];
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        data: { name: "New", description_html: "<p></p>" },
        onSubmit: async (value) => {
          submissions.push(value);
          if (submissions.length === 1)
            throw { custom_properties: { effort: "Use a positive number.", cost: "Enter cost." } };
        },
      })
    )
  );
  const input = customControl(env.container, "effort");
  await change(input, "-2");
  await submit(env.container);
  assert.equal(input.value, "-2");
  assert.equal(input.getAttribute("aria-invalid"), "true");
  assert.equal(document.activeElement, input);
  assert.match(document.getElementById(input.getAttribute("aria-describedby")).textContent, /positive/);
  await change(input, "2");
  assert.equal(document.activeElement, input, "correcting one field must not jump to the next error");
  assert.notEqual(input.getAttribute("aria-invalid"), "true");
  const cost = customControl(env.container, "cost");
  assert.equal(cost.getAttribute("aria-invalid"), "true", "the other field must retain its server error");
  await change(cost, "3");
  await submit(env.container);
  assert.equal(submissions.length, 2, "native field errors must permit submission after correction");
  assert.deepEqual(submissions[1].custom_properties, { effort: 2, cost: 3 });
});

await test("editing unrelated fields omits unchanged Member subscriptions from the payload", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("watchers", "Member picker", { multiple: true })]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  let saved;
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        data: {
          id: "existing",
          type_id: "task",
          project_id: "first",
          name: "Existing",
          description_html: "<p></p>",
          custom_properties: { watchers: ["member"] },
        },
        onSubmit: async (value) => {
          saved = value;
        },
      })
    )
  );
  await change(env.container.querySelector('[name="name"]'), "Edited title");
  await submit(env.container);
  assert.deepEqual(saved.custom_properties, {});
});

await test("draft publication reveals Cascading fields and publishes their retained selection", async (t) => {
  const env = await setup(t);
  const cascade = propertyDefinition("area", "Cascading", {
    levels: [{ id: "root", name: "Area", required: true }],
    options: [{ id: "platform", name: "Platform", parent: "", level: 0, next_label: "", icon: "" }],
  });
  enableTypeFields((input) => (input.mode === "publish" ? [cascade] : []));
  const moves = [];
  state.move = async (...args) => moves.push(args);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        isDraft: true,
        moveToIssue: true,
        data: { id: "draft", project_id: "first", type_id: "task", name: "Publish", description_html: "<p></p>" },
      })
    )
  );
  assert.equal(env.container.querySelector("select"), null);
  await act(async () =>
    [...env.container.querySelectorAll("button")].find((button) => button.textContent === "add_to_project").click()
  );
  await change(env.container.querySelector("select"), "platform");
  await act(async () =>
    [...env.container.querySelectorAll("button")].find((button) => button.textContent === "Publish work item").click()
  );
  assert.deepEqual(moves[0][2].custom_properties, { area: ["platform"] });
});

await test("Save to Drafts reloads draft schema and retains input while the creation schema is pending", async (t) => {
  const env = await setup(t);
  const effort = propertyDefinition("effort", "Number");
  const cascade = propertyDefinition("area", "Cascading", {
    levels: [{ id: "root", name: "Area", required: true }],
    options: [{ id: "platform", name: "Platform", parent: "", level: 0, next_label: "", icon: "" }],
  });
  enableTypeFields([effort, cascade]);
  const schema = state.schema;
  state.schema = async (input) => {
    if (input.mode === "draft") return { ...(await schema(input)), properties: [effort] };
    if (input.type_id === "bug") return new Promise(() => {});
    return schema(input);
  };
  const { DraftIssueLayout } = await import("../../apps/web/core/components/issues/issue-modal/draft-issue-layout.tsx");
  const saved = [];
  function Harness() {
    const [changesMade, setChanges] = React.useState(null);
    return React.createElement(DraftIssueLayout, {
      ...issueProps({
        data: { name: "Draft exit", description_html: "<p></p>" },
        onSubmit: async (value, draft) => saved.push({ value, draft }),
      }),
      changesMade,
      onChange: setChanges,
    });
  }
  await env.render(React.createElement(Harness));
  await change(customControl(env.container, "effort"), "0");
  await change(env.container.querySelector("select"), "platform");
  await chooseWorkItemType(env.container, "bug");
  await act(async () =>
    [...env.container.querySelectorAll("button")].find((button) => button.textContent === "discard").click()
  );
  await act(async () =>
    [...document.querySelectorAll("button")].find((button) => button.textContent === "Save to Drafts").click()
  );
  assert.equal(saved.length, 1);
  assert.equal(saved[0].draft, true);
  assert.deepEqual(saved[0].value.custom_properties, { effort: 0 });
});

async function waitForValue(predicate) {
  const deadline = Date.now() + 2000;
  const check = async () => {
    if (predicate() || Date.now() >= deadline) return;
    await act(async () => new Promise((resolve) => setTimeout(resolve, 10)));
    return check();
  };
  await check();
  assert.ok(predicate());
}
await test("formula preview omits an unchanged estimate but sends an explicit clear", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("calculated", "Formula", { expression: "1", result: "Number" })]);
  const previews = [];
  state.preview = async (_input, values) => {
    previews.push(values);
    return { computed: { calculated: { kind: "Number", status: "value", value: 0, error: "" } } };
  };
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        isDraft: true,
        data: {
          id: "draft",
          project_id: "first",
          type_id: "task",
          estimate_point: "private-estimate",
          name: "Private",
          description_html: "<p></p>",
        },
      })
    )
  );
  await waitForValue(() => previews.length > 0);
  assert.equal(Object.hasOwn(previews[0], "estimate_point"), false);
  const form = formObserver.getForms(formSources.issue).at(-1);
  await act(async () => form.setFieldValue("estimate_point", null));
  await waitForValue(() => previews.some((value) => value.estimate_point === ""));
  assert.equal(env.container.querySelector("output").textContent, "0");
});

await test("editing loads authoritative custom values and background refresh preserves a dirty field", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("effort", "Number")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const data = { id: "existing", project_id: "first", type_id: "task", name: "Existing", description_html: "<p></p>" };
  const render = (custom_properties) =>
    env.render(React.createElement(IssueFormRoot, issueProps({ data: { ...data, custom_properties } })));
  await render(undefined);
  await render({ effort: 7 });
  assert.equal(customControl(env.container, "effort").value, "7");
  await change(customControl(env.container, "effort"), "9");
  await render({ effort: 11 });
  assert.equal(customControl(env.container, "effort").value, "9");
});

await test("the modal source fetches custom values alongside description when opened from a compact item", async (t) => {
  const env = await setup(t);
  const { useIssueModalSource } =
    await import("../../apps/web/core/components/issues/issue-modal/use-issue-modal-source.ts");
  const fetched = { description_html: "<p>Full description</p>", type_id: "bug", custom_properties: { effort: 8 } };
  const { workItemEditsService } = await import("../../apps/web/core/services/work-item-edits.service.ts");
  t.mock.method(workItemEditsService, "source", async () => ({ item: fetched, source_version: "saved-v1" }));
  function Harness() {
    const source = useIssueModalSource({
      data: { id: "existing", project_id: "first" },
      isOpen: true,
      fetchIssueDetails: true,
      workspaceSlug: "workspace",
      allowedProjectIds: ["first"],
      getProjectByIdentifier: () => undefined,
    });
    return React.createElement("output", null, JSON.stringify(source.fields ?? null));
  }
  await env.render(React.createElement(Harness));
  assert.deepEqual(JSON.parse(env.container.querySelector("output").textContent), fetched);
});

await test("a saved projectless draft loads canonical fields even when ordinary detail fetching is disabled", async (t) => {
  const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
  t.mock.method(IssueService.prototype, "retrieve", () => {
    throw new Error("Drafts must not read the issue endpoint");
  });
  const env = await setup(t);
  const { workItemEditsService } = await import("../../apps/web/core/services/work-item-edits.service.ts");
  const { useIssueModalSource } =
    await import("../../apps/web/core/components/issues/issue-modal/use-issue-modal-source.ts");
  const fetched = {
    id: "draft",
    project_id: null,
    name: "",
    description_html: "<p>Private</p>",
    type_id: "bug",
    custom_properties: { effort: 0 },
    module_ids: ["saved"],
  };
  const { WorkspaceDraftService } = await import("../../apps/web/core/services/issue/workspace_draft.service.ts");
  t.mock.method(WorkspaceDraftService.prototype, "getIssueById", async () => ({
    ...fetched,
    created_by: "workspace-query-test-user",
  }));
  let request;
  t.mock.method(workItemEditsService, "source", async (...args) => {
    request = args;
    return { item: fetched, source_version: "draft-v1" };
  });
  function Harness() {
    const source = useIssueModalSource({
      data: { id: "draft", project_id: "stale" },
      isDraft: true,
      isOpen: true,
      fetchIssueDetails: false,
      workspaceSlug: "workspace",
      allowedProjectIds: ["first"],
      getProjectByIdentifier: () => undefined,
    });
    return React.createElement(
      "output",
      null,
      JSON.stringify({ fields: source.fields, project: source.activeProjectId })
    );
  }
  await env.render(React.createElement(Harness));
  assert.deepEqual(JSON.parse(env.container.querySelector("output").textContent), { fields: fetched, project: null });
  assert.deepEqual(request.slice(0, 4), ["workspace", "draft", "draft", undefined]);
  assert.equal(typeof request[4].addEventListener, "function");
});

await test("publication recovery renders the retained request without refetching a consumed draft", async (t) => {
  const env = await setup(t);
  const { workItemEditsService } = await import("../../apps/web/core/services/work-item-edits.service.ts");
  const { useIssueModalSource } =
    await import("../../apps/web/core/components/issues/issue-modal/use-issue-modal-source.ts");
  const read = t.mock.method(workItemEditsService, "source", async () => {
    throw new Error("Consumed draft");
  });
  const retained = { project_id: "destination", name: "Published", description_html: "<p>Original</p>" };
  function Harness() {
    const source = useIssueModalSource({
      data: { id: "draft" },
      isDraft: true,
      retainedPublication: retained,
      isOpen: true,
      fetchIssueDetails: true,
      workspaceSlug: "workspace",
      allowedProjectIds: [],
      getProjectByIdentifier: () => undefined,
    });
    return React.createElement(
      "output",
      null,
      JSON.stringify({ fields: source.fields, project: source.activeProjectId, description: source.description })
    );
  }
  await env.render(React.createElement(Harness));
  assert.deepEqual(JSON.parse(env.container.querySelector("output").textContent), {
    fields: retained,
    project: "destination",
    description: retained.description_html,
  });
  assert.equal(read.mock.callCount(), 0);
});
await test("a refreshed untouched type is hydrated without resubmitting untouched concurrent custom changes", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("effort", "Number"), propertyDefinition("cost", "Number")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  let saved;
  const data = { id: "existing", project_id: "first", name: "Existing", description_html: "<p></p>" };
  const render = (type_id, custom_properties) =>
    env.render(
      React.createElement(
        IssueFormRoot,
        issueProps({
          data: { ...data, type_id, custom_properties },
          onSubmit: async (value) => {
            saved = value;
          },
        })
      )
    );
  await render("task", { effort: 1, cost: 1 });
  await render("bug", { effort: 1, cost: 1 });
  assert.equal(env.container.querySelector('[aria-label="Work item type"]').textContent, "bug");
  await change(customControl(env.container, "effort"), "2");
  await render("bug", { effort: 1, cost: 3 });
  await submit(env.container);
  assert.equal(Object.hasOwn(saved, "type_id"), false);
  assert.deepEqual(saved.custom_properties, { effort: 2 });
});
await test("rich text accepts pristine hydration without rewriting local keystrokes", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("notes", "Rich text")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const data = { id: "existing", project_id: "first", type_id: "task", name: "Existing", description_html: "<p></p>" };
  const render = (notes) =>
    env.render(React.createElement(IssueFormRoot, issueProps({ data: { ...data, custom_properties: { notes } } })));
  await render("");
  await render("<p>Fetched</p>");
  assert.equal(customControl(env.container, "notes").value, "<p>Fetched</p>");
  const writes = state.editorWrites.length;
  await change(customControl(env.container, "notes"), "<p>Local</p>");
  await render("<p>Concurrent</p>");
  assert.equal(customControl(env.container, "notes").value, "<p>Local</p>");
  assert.equal(state.editorWrites.length, writes, "typing never resets editor content");
});

await test("a busy rich-text editor defers refresh and blocks scope changes until ready", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("notes", "Rich text")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const data = { project_id: "first", type_id: "task", name: "New", description_html: "<p></p>" };
  const render = (notes) =>
    env.render(React.createElement(IssueFormRoot, issueProps({ data: { ...data, custom_properties: { notes } } })));
  await render("<p>Initial</p>");
  state.editorReady = false;
  await render("<p>Refreshed</p>");
  assert.equal(customControl(env.container, "notes").value, "<p>Initial</p>");
  await chooseWorkItemType(env.container, "bug");
  assert.equal(env.container.querySelector('[aria-label="Work item type"]').textContent, "task");
  await change(env.container.querySelector('[name="project_id"]'), "second");
  assert.equal(env.container.querySelector('[name="project_id"]').value, "first");
  await act(async () => {
    state.editorReady = true;
    for (const editor of state.editors.values()) editor.emit();
  });
  assert.equal(customControl(env.container, "notes").value, "<p>Refreshed</p>");
  state.editorReady = false;
  await render("<p>Stale incoming</p>");
  await change(customControl(env.container, "notes"), "<p>New local</p>");
  await act(async () => {
    state.editorReady = true;
    for (const editor of state.editors.values()) editor.emit();
  });
  assert.equal(customControl(env.container, "notes").value, "<p>New local</p>");
});
await test("temporary schema failures retain the mounted rich text editor but access denial hides it", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("notes", "Rich text")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        data: { name: "New", project_id: "first", type_id: "task", custom_properties: { notes: "Retain" } },
      })
    )
  );
  const control = customControl(env.container, "notes");
  state.editorReady = false;
  const verifyTemporaryFailure = async (status) => {
    state.schema = async () => {
      throw { isAxiosError: true, response: { status } };
    };
    await act(async () => env.client.invalidateQueries({ queryKey: ["workspace", "workspace", "work-item-schema"] }));
    assert.equal(customControl(env.container, "notes"), control);
    assert.match(env.container.textContent, /Could not load work item fields/);
  };
  await verifyTemporaryFailure(429);
  await verifyTemporaryFailure(408);
  await verifyTemporaryFailure(503);
  state.schema = async () => {
    throw { isAxiosError: true, response: { status: 403 } };
  };
  await act(async () => env.client.invalidateQueries({ queryKey: ["workspace", "workspace", "work-item-schema"] }));
  assert.equal(control.isConnected, false);
});

await test("rich text validation focuses its editable surface and exposes the field label and error", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("notes", "Rich text")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        data: { name: "New", description_html: "<p></p>", project_id: "first", type_id: "task" },
        onSubmit: async () => {
          throw { custom_properties: { notes: "Enter notes." } };
        },
      })
    )
  );
  await submit(env.container);
  const control = customControl(env.container, "notes");
  assert.ok(
    document.activeElement === control,
    `expected rich editor focus, got ${document.activeElement?.tagName}#${document.activeElement?.id}`
  );
  assert.equal(control.getAttribute("aria-invalid"), "true");
  assert.match(document.getElementById(control.getAttribute("aria-labelledby")).textContent, /notes/);
  assert.match(document.getElementById(control.getAttribute("aria-describedby")).textContent, /Enter notes/);
});
await test("create-more displays the exact retained rich text values submitted for the next item", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("notes", "Rich text")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const saved = [];
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        isCreateMoreToggleEnabled: true,
        data: {
          name: "Template",
          description_html: "<p>Template description</p>",
          project_id: "first",
          type_id: "task",
          custom_properties: { notes: "<p>Template notes</p>" },
        },
        onSubmit: async (value) => saved.push(value),
      })
    )
  );
  await submit(env.container);
  assert.equal(customControl(env.container, "notes").value, "<p>Template notes</p>");
  assert.equal(env.container.querySelector('[name="description_html"]').value, "<p>Template description</p>");
  await change(customControl(env.container, "notes"), "<p>Second notes</p>");
  await submit(env.container);
  assert.equal(saved[1].custom_properties.notes, "<p>Second notes</p>");
  assert.equal(customControl(env.container, "notes").value, "<p>Template notes</p>");
});
await test("refresh waits for every rich text upload before accepting new saved values", async (t) => {
  const env = await setup(t);
  enableTypeFields([propertyDefinition("notes", "Rich text"), propertyDefinition("details", "Rich text")]);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const render = (custom_properties) =>
    env.render(
      React.createElement(
        IssueFormRoot,
        issueProps({ data: { id: "item", project_id: "first", type_id: "task", name: "Item", custom_properties } })
      )
    );
  await render({ notes: "Old notes", details: "Old details" });
  const notes = customControl(env.container, "notes");
  const details = customControl(env.container, "details");
  state.editorReadiness.set(notes.id, false);
  state.editorReadiness.set(details.id, false);
  await render({ notes: "New notes", details: "New details" });
  await act(async () => {
    state.editorReadiness.set(notes.id, true);
    state.editors.get(notes.id).emit();
  });
  assert.equal(notes.value, "Old notes");
  assert.equal(details.value, "Old details");
  await act(async () => {
    state.editorReadiness.set(details.id, true);
    state.editors.get(details.id).emit();
  });
  assert.equal(notes.value, "New notes");
  assert.equal(details.value, "New details");
});

await test("an invalid type after a project change still allows choosing an eligible replacement", async (t) => {
  const env = await setup(t);
  enableTypeFields([]);
  const schema = state.schema;
  state.schema = async (input) =>
    input.project_id === "second" && input.type_id !== "bug"
      ? {
          ...(await schema(input)),
          type_id: "",
          eligible_type_ids: ["bug"],
          selection_reason: "Choose a type available in this project.",
        }
      : schema(input);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({ data: { name: "New", description_html: "<p></p>", project_id: "first", type_id: "task" } })
    )
  );
  await change(env.container.querySelector('[name="project_id"]'), "second");
  assert.match(env.container.querySelector('[aria-label="Work item type"]').textContent, /task/);
  await chooseWorkItemType(env.container, "bug");
  assert.match(env.container.querySelector('[aria-label="Work item type"]').textContent, /bug/);
});

await test("saved untyped drafts retain their empty choice and omit an unchanged masked parent", async (t) => {
  const env = await setup(t);
  enableTypeFields([]);
  const schema = state.schema;
  const scopes = [];
  state.schema = async (input) => {
    scopes.push(input);
    return { ...(await schema(input)), type_id: "", selection_reason: "Choose a work item type." };
  };
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        isDraft: true,
        data: {
          id: "draft",
          name: "Untyped",
          description_html: "<p></p>",
          project_id: "first",
          type_id: null,
          parent_id: null,
        },
      })
    )
  );
  assert.equal(env.container.querySelector('[aria-label="Work item type"]').textContent, "Select type");
  assert.ok(scopes.length > 0);
  assert.ok(
    scopes.every((scope) => !Object.hasOwn(scope, "parent_id")),
    "masked parent omission must not turn into an explicit clear"
  );
  await chooseWorkItemType(env.container, "bug");
  assert.match(env.container.querySelector('[aria-label="Work item type"]').textContent, /bug/);
});

await test("a late default suggestion never replaces a newly explicit type choice", async (t) => {
  const env = await setup(t);
  enableTypeFields([]);
  const schema = state.schema;
  let finish;
  state.schema = async (input) =>
    input.type_id
      ? schema(input)
      : new Promise((resolve) => {
          finish = async () => resolve(await schema(input));
        });
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  formObserver.clear();
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({ data: { name: "New", description_html: "<p></p>", project_id: "first" } })
    )
  );
  const form = formObserver.getForms(formSources.issue).at(-1);
  assert.ok(finish);
  await act(async () => form.setFieldValue("type_id", "bug"));
  await act(async () => finish());
  assert.equal(form.state.values.type_id, "bug");
  assert.match(env.container.querySelector('[aria-label="Work item type"]').textContent, /bug/);
});

await test("reopening creation waits for the refreshed project default instead of caching a choice", async (t) => {
  const env = await setup(t);
  enableTypeFields([]);
  const schema = state.schema;
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const render = () =>
    env.render(
      React.createElement(
        IssueFormRoot,
        issueProps({ data: { name: "New", description_html: "<p></p>", project_id: "first" } })
      )
    );
  await render();
  assert.match(env.container.querySelector('[aria-label="Work item type"]').textContent, /task/);
  await env.render(null);
  let finish;
  state.schema = async (input) =>
    input.type_id
      ? schema(input)
      : new Promise((resolve) => {
          finish = async () => resolve({ ...(await schema(input)), type_id: "bug" });
        });
  await render();
  assert.ok(finish);
  assert.equal(env.container.querySelector('[aria-label="Work item type"]').textContent, "Select type");
  await act(async () => finish());
  assert.match(env.container.querySelector('[aria-label="Work item type"]').textContent, /bug/);
});
formObserver.restore();

await test("copying a compact issue reads its description and custom values with cancellation", async (t) => {
  const env = await setup(t);
  const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
  const { useIssueModalSource } =
    await import("../../apps/web/core/components/issues/issue-modal/use-issue-modal-source.ts");
  const fetched = {
    id: "original",
    project_id: "first",
    description_html: "<p>Original description</p>",
    type_id: "bug",
    custom_properties: { effort: 8 },
  };
  const read = t.mock.method(IssueService.prototype, "retrieve", async () => fetched);
  function Harness() {
    const source = useIssueModalSource({
      data: { sourceIssueId: "original", project_id: "first" },
      isOpen: true,
      fetchIssueDetails: true,
      workspaceSlug: "workspace",
      getProjectByIdentifier: () => undefined,
    });
    return React.createElement("output", null, JSON.stringify(source.fields ?? null));
  }
  await env.render(React.createElement(Harness));
  assert.deepEqual(JSON.parse(env.container.querySelector("output").textContent), {
    description_html: fetched.description_html,
    type_id: "bug",
    custom_properties: { effort: 8 },
  });
  assert.deepEqual(read.mock.calls[0].arguments.slice(0, 3), ["workspace", "first", "original"]);
  assert.equal(read.mock.calls[0].arguments[4].signal.aborted, false);
});

await test("copying an issue rejects a description response from a different project", async (t) => {
  const env = await setup(t);
  const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
  const { useIssueModalSource } =
    await import("../../apps/web/core/components/issues/issue-modal/use-issue-modal-source.ts");
  t.mock.method(IssueService.prototype, "retrieve", async () => ({
    id: "original",
    project_id: "second",
    description_html: "Foreign source",
  }));
  function Harness() {
    const source = useIssueModalSource({
      data: { sourceIssueId: "original", project_id: "first" },
      isOpen: true,
      fetchIssueDetails: true,
      workspaceSlug: "workspace",
      getProjectByIdentifier: () => undefined,
    });
    return React.createElement(
      "output",
      null,
      JSON.stringify({ status: source.descriptionQuery.status, fields: source.fields })
    );
  }
  await env.render(React.createElement(Harness));
  assert.deepEqual(JSON.parse(env.container.querySelector("output").textContent), { status: "error" });
});

import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl as change, submitForm as submit, waitForSubmitCompletion } from "./helpers/native-forms.mjs";
import { setup, state, issueProps, formObserver, formSources } from "./helpers/issue-form-harness.mjs";

await test("changing a work item's project clears project-specific properties but preserves written content", async (t) => {
  const env = await setup(t);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  let saved;
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        data: {
          name: "Draft",
          description_html: "<p>Original</p>",
          state_id: "old-state",
          assignee_ids: ["old-member"],
          label_ids: ["old-label"],
          cycle_id: "old-cycle",
          module_ids: ["old-module"],
        },
        onSubmit: async (value) => {
          saved = value;
        },
      })
    )
  );
  await change(env.container.querySelector('[name="name"]'), "Keep this title");
  await change(env.container.querySelector('[name="description_html"]'), "<p>Keep this description</p>");
  await change(env.container.querySelector('[name="project_id"]'), "second");
  await submit(env.container);
  assert.equal(saved.project_id, "second");
  assert.equal(saved.name, "Keep this title");
  assert.equal(saved.description_html, "<p>Keep this description</p>");
  assert.equal(saved.state_id, "");
  assert.equal(saved.cycle_id, null);
  for (const field of ["assignee_ids", "label_ids", "module_ids"]) assert.deepEqual(saved[field], []);
});
await test("failed work-item save keeps edits and blocks closing while pending", async (t) => {
  const env = await setup(t);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  let reject,
    closed = 0;
  const props = issueProps({
    onClose: () => closed++,
    onSubmit: () =>
      new Promise((_resolve, rejectSave) => {
        reject = rejectSave;
      }),
  });
  await env.render(React.createElement(IssueFormRoot, props));
  await change(env.container.querySelector('[name="name"]'), "Unsaved title");
  await submit(env.container);
  assert.ok(env.container.querySelector("fieldset").disabled);
  await act(async () =>
    [...env.container.querySelectorAll("button")].find((button) => button.textContent === "discard").click()
  );
  assert.equal(closed, 0);
  await act(async () => reject(new Error("offline")));
  assert.equal(env.container.querySelector('[name="name"]').value, "Unsaved title");
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Failed to save/);
  assert.equal(formObserver.getForms(formSources.issue).at(-1)?.state.isSubmitSuccessful, false);
  assert.equal(env.container.querySelector("fieldset").disabled, false);
});
await test("moving a draft validates the edited title before sending it", async (t) => {
  const env = await setup(t);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const moves = [];
  state.move = async (...args) => moves.push(args);
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        isDraft: true,
        moveToIssue: true,
        data: { id: "draft", project_id: "first", description_html: "<p>Draft</p>" },
      })
    )
  );
  const move = [...env.container.querySelectorAll("button")].find((button) => button.textContent === "add_to_project");
  await act(async () => move.click());
  assert.equal(moves.length, 0);
  await change(env.container.querySelector('[name="name"]'), "Valid draft");
  await act(async () => move.click());
  assert.equal(moves.length, 1);
  assert.equal(moves[0][2].name, "Valid draft");
});
await test("work-item submission stays unsuccessful while editor uploads are pending and retries when ready", async (t) => {
  const env = await setup(t);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  const saved = [];
  state.editorReady = false;
  await env.render(React.createElement(IssueFormRoot, issueProps({ onSubmit: async (value) => saved.push(value) })));
  await change(env.container.querySelector('[name="name"]'), "Keep until uploads finish");
  const button = env.container.querySelector('button[type="submit"]');
  await act(async () => button.click());
  const formApi = formObserver.getForms(formSources.issue).at(-1);
  assert.deepEqual(saved, []);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(env.container.querySelector('[name="name"]').value, "Keep until uploads finish");
  state.editorReady = true;
  assert.equal(button.disabled, false);
  await act(async () => button.click());
  assert.equal(saved.length, 1);
  assert.equal(formApi.state.isSubmitSuccessful, true);
});
await test("retrying an unfinished creation reuses its issue and completed durable steps", async (t) => {
  const env = await setup(t);
  const { useIssueCreation } = await import("../../apps/web/core/components/issues/issue-modal/use-issue-creation.ts");
  const calls = [];
  const moduleRequests = [];
  const request = { payload: { project_id: "first", name: "New issue", module_ids: ["original"] }, isDraft: false };
  let fail = true;
  function Harness() {
    const [error, setError] = React.useState("");
    const [failure, setFailure] = React.useState(null);
    const workflow = useIssueCreation({
      create: async () => {
        calls.push("create");
        return { id: "created", project_id: "first" };
      },
      finalizeAssets: async () => {
        calls.push("assets");
      },
      assignCycle: async () => {
        calls.push("cycle");
      },
      assignModules: async (_issue, submitted) => {
        calls.push("modules");
        moduleRequests.push(structuredClone(submitted));
        if (fail) throw new Error("offline");
      },
      onSaved: async (issue) => {
        calls.push("saved:" + issue.id);
      },
    });
    return React.createElement(
      "button",
      {
        "data-failure-name": failure?.name,
        "data-failure-id": failure?.issueId,
        "data-failure-cause": failure?.cause?.message,
        onClick: async () => {
          try {
            await workflow.submit(request);
            setError("Saved");
          } catch (errorCause) {
            setFailure(errorCause);
            setError(errorCause.message);
          }
        },
      },
      error || "Save"
    );
  }
  await env.render(React.createElement(Harness));
  await act(async () => {
    env.container.querySelector("button").click();
  });
  const button = env.container.querySelector("button");
  assert.equal(button.dataset.failureName, "IssueCreationIncompleteError");
  assert.equal(button.dataset.failureId, "created", "the accepted entity ID survives the assignment failure");
  assert.equal(button.dataset.failureCause, "offline");
  assert.match(env.container.querySelector("button").textContent, /created.*Retry to finish/);
  fail = false;
  request.payload.module_ids[0] = "changed";
  await act(async () => {
    env.container.querySelector("button").click();
  });
  assert.deepEqual(calls, ["create", "assets", "cycle", "modules", "modules", "saved:created"]);
  assert.deepEqual(
    moduleRequests.map((entry) => entry.payload.module_ids),
    [["original"], ["original"]]
  );
});
await test("project creation retries cover finalization without creating another project", async (t) => {
  const env = await setup(t);
  const { CreateProjectForm } = await import("../../apps/web/core/components/projects/create/root.tsx");
  const created = [],
    next = [],
    finalized = [];
  let fail = true;
  const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
  t.mock.method(ProjectService.prototype, "createProject", async (_workspace, value) => {
    created.push(value);
    return { ...value, id: "project", workspace: "workspace" };
  });
  await env.render(
    React.createElement(CreateProjectForm, {
      workspaceSlug: "workspace",
      onClose: () => {},
      handleNextStep: (id) => next.push(id),
      updateCoverImageStatus: async (id) => {
        finalized.push(id);
        if (fail) throw new Error("offline");
      },
    })
  );
  await change(document.querySelector('[name="name"]'), "Example Project");
  assert.equal(document.querySelector('[name="identifier"]').value, "EXAMPLEPRO");
  await change(document.querySelector('[name="identifier"]'), "custom");
  await change(document.querySelector('[name="name"]'), "Renamed Project");
  await submit(document);
  await act(async () => waitForSubmitCompletion(formObserver.getForms(formSources.project).at(-1)));
  assert.equal(created[0].identifier, "CUSTOM");
  assert.equal(next.length, 0);
  assert.equal(formObserver.getForms(formSources.project).at(-1)?.state.isSubmitSuccessful, false);
  fail = false;
  await submit(document);
  await act(async () => waitForSubmitCompletion(formObserver.getForms(formSources.project).at(-1)));
  assert.equal(created.length, 1);
  assert.deepEqual(finalized, ["project", "project"]);
  assert.deepEqual(next, ["project"]);
});

await test("quick add keeps a failed draft and clears it only after creation succeeds", async (t) => {
  const env = await setup(t);
  const { QuickAddIssueForm } =
    await import("../../apps/web/core/components/issues/issue-layouts/quick-add/form/root.tsx");
  let fail = true;
  const calls = [];
  await env.render(
    React.createElement(QuickAddIssueForm, {
      workspaceSlug: "workspace",
      projectId: "first",
      layout: "gantt_chart",
      isEpic: false,
      onClose: () => {},
      onCreate: async (id, value) => {
        calls.push({ id, value });
        if (fail) throw new Error("offline");
        return { ...value, id: "new" };
      },
    })
  );
  await change(env.container.querySelector("input"), "Quick draft");
  await submit(env.container);
  assert.equal(env.container.querySelector("input").value, "Quick draft");
  assert.equal(formObserver.getForms(formSources.quickAdd).at(-1)?.state.isSubmitSuccessful, false);
  fail = false;
  await submit(env.container);
  assert.equal(calls[1].value.name, "Quick draft");
  assert.equal(env.container.querySelector("input").value, "");
});
await test("intake validation prevents empty titles and preserves failed submissions", async (t) => {
  const env = await setup(t);
  const { InboxIssueCreateRoot } =
    await import("../../apps/web/core/components/inbox/modals/create-modal/create-root.tsx");
  let creates = 0;
  state.intake = {
    create: async () => {
      creates++;
      throw new Error("offline");
    },
  };
  await env.render(
    React.createElement(InboxIssueCreateRoot, {
      workspaceSlug: "workspace",
      projectId: "first",
      handleModalClose: () => {},
    })
  );
  await submit(env.container);
  assert.equal(creates, 0);
  await change(env.container.querySelector('[name="name"]'), "Intake draft");
  await submit(env.container);
  assert.equal(creates, 1);
  assert.equal(env.container.querySelector('[name="name"]').value, "Intake draft");
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Could not finish/);
  assert.equal(formObserver.getForms(formSources.intake).at(-1)?.state.isSubmitSuccessful, false);
});
await test("intake submission waits for editor uploads and retries the retained draft", async (t) => {
  const env = await setup(t);
  const { InboxIssueCreateRoot } =
    await import("../../apps/web/core/components/inbox/modals/create-modal/create-root.tsx");
  const saved = [];
  state.editorReady = false;
  state.intake = {
    create: async (_workspace, _project, value) => {
      saved.push(value);
      return { issue: { id: "created" } };
    },
  };
  await env.render(
    React.createElement(InboxIssueCreateRoot, {
      workspaceSlug: "workspace",
      projectId: "first",
      handleModalClose: () => {},
    })
  );
  await change(env.container.querySelector('[name="name"]'), "Intake upload draft");
  const button = env.container.querySelector('button[type="submit"]');
  await act(async () => button.click());
  const formApi = formObserver.getForms(formSources.intake).at(-1);
  assert.deepEqual(saved, []);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(env.container.querySelector('[name="name"]').value, "Intake upload draft");
  state.editorReady = true;
  assert.equal(button.disabled, false);
  await act(async () => button.click());
  assert.equal(saved.length, 1);
  assert.equal(formApi.state.isSubmitSuccessful, true);
});
await test("page creation retains the user's draft when saving fails", async (t) => {
  const env = await setup(t);
  const { PageForm } = await import("../../apps/web/core/components/pages/modals/page-form.tsx");
  const calls = [];
  await env.render(
    React.createElement(PageForm, {
      handleModalClose: () => {},
      handleFormSubmit: async (value) => {
        calls.push(value);
        throw new Error("offline");
      },
    })
  );
  await submit(document);
  assert.equal(calls.length, 0);
  await change(document.querySelector('[name="name"]'), "Page draft");
  await submit(document);
  assert.equal(calls[0].name, "Page draft");
  assert.equal(document.querySelector('[name="name"]').value, "Page draft");
  assert.match(document.querySelector('[role="alert"]').textContent, /Could not create/);
  assert.equal(formObserver.getForms(formSources.page).at(-1)?.state.isSubmitSuccessful, false);
});

async function openEstimateDraft(env, system = "points") {
  const { CreateEstimateModal } = await import("../../apps/web/core/components/estimates/create/modal.tsx");
  await env.render(
    React.createElement(CreateEstimateModal, {
      workspaceSlug: "workspace",
      projectId: "first",
      isOpen: true,
      handleClose: () => {},
    })
  );
  await act(async () => document.querySelector(`input[value="${system}"]`).click());
  await act(async () =>
    [...document.querySelectorAll("button")]
      .find((button) => button.textContent.includes("project_settings.estimates.create.custom"))
      .click()
  );
}
await test("estimate values reject empty, nonnumeric, and duplicate values before saving", async (t) => {
  const env = await setup(t);
  const saved = [];
  state.createEstimate = async (...args) => saved.push(args);
  await openEstimateDraft(env);
  const first = document.querySelector('[aria-label="Estimate value 1"]');
  await change(first, "");
  await submit(document);
  assert.equal(saved.length, 0);
  await change(first, "abc");
  await submit(document);
  assert.equal(saved.length, 0);
  await change(first, " 2 ");
  await submit(document);
  assert.equal(saved.length, 0);
  assert.match(document.body.textContent, /already_exists/);
  await change(first, "3");
  await submit(document);
  assert.deepEqual(
    saved[0][2].estimate_points.map((point) => point.value),
    ["3", "2"]
  );
});
await test("estimate categories detect duplicates after trimming and preserve a failed draft", async (t) => {
  const env = await setup(t);
  const saved = [];
  state.createEstimate = async (_workspace, _project, value) => {
    saved.push(value);
    throw new Error("offline");
  };
  await openEstimateDraft(env, "categories");
  const first = document.querySelector('[aria-label="Estimate value 1"]');
  await change(first, " Hard ");
  await submit(document);
  assert.equal(saved.length, 0);
  await change(first, " Simple ");
  await submit(document);
  assert.equal(saved[0].estimate_points[0].value, "Simple");
  assert.equal(first.value, " Simple ");
  assert.match(document.body.textContent, /created.error.message/);
  assert.equal(formObserver.getForms(formSources.estimate).at(-1)?.state.isSubmitSuccessful, false);
});
await test("estimate array removal clears the removed field's validation and renumbers saved values", async (t) => {
  const env = await setup(t);
  const saved = [];
  state.createEstimate = async (_workspace, _project, value) => saved.push(value);
  await openEstimateDraft(env);
  assert.equal(document.querySelector('[aria-label="Remove estimate 1"]').disabled, true);
  await act(async () =>
    [...document.querySelectorAll("button")].find((button) => button.textContent === "Add points").click()
  );
  await submit(document);
  assert.equal(saved.length, 0);
  await act(async () => document.querySelector('[aria-label="Remove estimate 3"]').click());
  await submit(document);
  assert.deepEqual(saved[0].estimate_points, [
    { key: 1, value: "1", id: undefined },
    { key: 2, value: "2", id: undefined },
  ]);
});

await test("state forms validate blank names and retain drafts after failure while locking pending edits", async (t) => {
  const env = await setup(t);
  const { StateForm } = await import("../../apps/web/core/components/project-states/create-update/form.tsx");
  let reject;
  let saved;
  await env.render(
    React.createElement(StateForm, {
      data: { name: "", description: "Initial description", color: "#112233" },
      buttonTitle: "Create",
      onCancel: () => {},
      onSubmit: (value) => {
        saved = value;
        return new Promise((_resolve, rejectSave) => {
          reject = rejectSave;
        });
      },
    })
  );
  await change(env.container.querySelector('[name="name"]'), "   ");
  await submit(env.container);
  assert.equal(saved, undefined);
  await change(env.container.querySelector('[name="name"]'), " Ready ");
  await change(env.container.querySelector('[name="description"]'), "Keep this draft");
  await submit(env.container);
  assert.equal(saved.name, "Ready");
  assert.equal(saved.description, "Keep this draft");
  assert.equal(env.container.querySelector("fieldset").disabled, true);
  await act(async () => reject(new Error("Save failed")));
  assert.equal(env.container.querySelector('[name="description"]').value, "Keep this draft");
  assert.equal(env.container.querySelector("fieldset").disabled, false);
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Save failed/);
  assert.equal(formObserver.getForms(formSources.state).at(-1)?.state.isSubmitSuccessful, false);
});
await test("intake snooze retains the chosen date after failure and closes only after a successful retry", async (t) => {
  const env = await setup(t);
  const { InboxIssueSnoozeModal } = await import("../../apps/web/core/components/inbox/modals/snooze-issue-modal.tsx");
  const selected = new Date(2030, 0, 15);
  let fail = true;
  let closed = 0;
  const saved = [];
  await env.render(
    React.createElement(InboxIssueSnoozeModal, {
      isOpen: true,
      value: selected,
      handleClose: () => closed++,
      onConfirm: async (date) => {
        saved.push(date);
        if (fail) throw new Error("offline");
      },
    })
  );
  await submit(document);
  assert.equal(closed, 0);
  assert.match(document.querySelector('[role="alert"]').textContent, /Could not snooze/);
  assert.equal(formObserver.getForms(formSources.snooze).at(-1)?.state.isSubmitSuccessful, false);
  fail = false;
  await submit(document);
  assert.equal(closed, 1);
  assert.equal(saved[1].getTime(), selected.getTime());
});

await test("invitation selection validates and retries unfinished setup without accepting invitations twice", async (t) => {
  const env = await setup(t);
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const { InvitationSelectionForm } =
    await import("../../apps/web/core/components/workspace/invitation-selection-form.tsx");
  const accepted = [];
  t.mock.method(WorkspaceService.prototype, "joinWorkspaces", async (payload) => {
    accepted.push(payload);
  });
  let fail = true;
  const finished = [];
  await env.render(
    React.createElement(InvitationSelectionForm, {
      invitations: [{ id: "invite", role: 20, workspace: { id: "workspace", name: "Engineering" } }],
      onJoined: async (ids) => {
        if (fail) throw new Error("offline");
        finished.push(ids);
      },
      onCancel: () => {},
      submitLabel: "Join",
      cancelLabel: "Cancel",
    })
  );
  await submit(env.container);
  assert.deepEqual(accepted, []);
  const checkbox = env.container.querySelector('input[type="checkbox"]');
  await act(async () => checkbox.click());
  assert.equal(checkbox.checked, true);
  await act(async () => checkbox.click());
  assert.equal(checkbox.checked, false);
  await act(async () => checkbox.click());
  await submit(env.container);
  assert.equal(env.container.querySelector("fieldset").disabled, true);
  assert.match(env.container.querySelector('[role="alert"]').textContent, /Could not finish/);
  assert.equal(formObserver.getForms(formSources.invitation).at(-1)?.state.isSubmitSuccessful, false);
  fail = false;
  await submit(env.container);
  assert.deepEqual(accepted, [{ invitations: ["invite"] }]);
  assert.deepEqual(finished, [["invite"]]);
});

formObserver.restore();

await test("a projectless draft permits an empty title and keeps description content", async (t) => {
  const env = await setup(t);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  let saved;
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        isDraft: true,
        projectId: null,
        data: { project_id: null, name: "", description_html: "<p>Private note</p>" },
        onSubmit: async (payload, draft) => {
          saved = { payload, draft };
        },
      })
    )
  );
  assert.ok(env.container.querySelector('[name="description_html"]'));
  await submit(env.container);
  assert.equal(saved.draft, true);
  assert.equal(saved.payload.project_id, null);
  assert.equal(saved.payload.name, "");
  assert.equal(saved.payload.description_html, "<p>Private note</p>");
});

await test("a saved draft can change project without discarding its existing selections", async (t) => {
  const env = await setup(t);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  let saved;
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        isDraft: true,
        data: {
          id: "draft",
          project_id: "first",
          name: "Private",
          description_html: "<p>Keep me</p>",
          state_id: "old-state",
          module_ids: ["old-module"],
          label_ids: ["old-label"],
        },
        onSubmit: async (payload) => {
          saved = payload;
        },
      })
    )
  );
  const selector = env.container.querySelector('[name="project_id"]');
  assert.equal(selector.disabled, false);
  await change(selector, "second");
  assert.equal(env.container.querySelector('[name="module_ids"]').value, "old-module");
  await submit(env.container);
  assert.equal(saved.project_id, "second");
  assert.equal(Object.hasOwn(saved, "module_ids"), false, "untouched selections remain on the saved draft");
  assert.equal(Object.hasOwn(saved, "description_html"), false, "unchanged description is not overwritten");
});

await test("publication errors appear beside the selection and focus it without losing the draft", async (t) => {
  const env = await setup(t);
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await env.render(
    React.createElement(
      IssueFormRoot,
      issueProps({
        isDraft: true,
        moveToIssue: true,
        data: {
          id: "draft",
          project_id: "first",
          name: "Private",
          description_html: "<p>Keep</p>",
          module_ids: ["old-module"],
        },
        onPublish: async () => {
          throw { response: { status: 400, data: { module_ids: ["Choose an available module."] } } };
        },
      })
    )
  );
  const button = [...env.container.querySelectorAll("button")].find((el) => el.textContent === "add_to_project");
  await act(async () => button.click());
  await act(async () => button.click());
  const field = env.container.querySelector('[data-issue-field="module_ids"]');
  assert.match(field.textContent, /Choose an available module/);
  assert.equal(document.activeElement, field.querySelector("input"));
  assert.equal(field.querySelector("input").value, "old-module");
});

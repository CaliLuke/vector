import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { issueProps, setup, state } from "./helpers/issue-form-harness.mjs";

function ModalHarness({ children }) {
  const [isOpen, setIsOpen] = React.useState(true);
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("output", { "data-testid": "modal-open" }, String(isOpen)),
    isOpen && children(() => setIsOpen(false))
  );
}

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@/components/dropdowns/member/dropdown") {
      const source = `
        const React = globalThis.issueFormReact;
        export const MemberDropdown = () => {
          const [isOpen, setIsOpen] = React.useState(false);
          const input = isOpen ? React.createElement("input", {
            placeholder: "search",
            role: "combobox",
            onKeyDown: (event) => {
              if (event.key !== "Escape") return;
              event.preventDefault();
              setIsOpen(false);
            },
          }) : null;
          return React.createElement(React.Fragment, null,
            React.createElement("button", { type: "button", onClick: () => setIsOpen(true) }, "assignees"),
            input && globalThis.issueFormCreatePortal(input, document.body)
          );
        };
      `;
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(source)}` };
    }
    if (
      specifier === "@/components/core/modals/gpt-assistant-popover" &&
      context.parentURL?.includes("issue-modal/components/description-editor.tsx")
    ) {
      const source = `
        const React = globalThis.issueFormReact;
        export const GptAssistantPopover = ({ onOpen }) => React.createElement(
          "button", { type: "button", onClick: onOpen }, "open-assistant"
        );
      `;
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(source)}` };
    }
    return nextResolve(specifier, context);
  },
});

async function renderIssueForm(t, propsOverrides = {}, prepareEnvironment = () => {}) {
  const environment = await setup(t);
  prepareEnvironment(environment);
  globalThis.issueFormCreatePortal = webRequire("react-dom").createPortal;
  const { IssueFormRoot } = await import("../../apps/web/core/components/issues/issue-modal/form.tsx");
  await environment.render(
    React.createElement(ModalHarness, null, (onClose) =>
      React.createElement(
        IssueFormRoot,
        issueProps({
          data: { name: "Create item", description_html: "<p></p>", project_id: "first", type_id: "task" },
          onClose,
          ...propsOverrides,
        })
      )
    )
  );
  return environment;
}

function pressEscape(element) {
  element.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
}

async function openAssigneePicker() {
  const trigger = [...document.querySelectorAll("button")].find((button) => button.textContent.includes("assignees"));
  assert.ok(trigger, "the issue form should render its Assignees control");
  await act(async () => trigger.click());
  const input = document.querySelector('input[placeholder="search"]');
  assert.ok(input, "opening Assignees should mount its search input");
  await act(async () => input.focus());
  assert.equal(document.activeElement, input, "the picker should focus its search input");
  return input;
}

await test("assignee Escape closes the picker before the work-item form", async (t) => {
  const environment = await renderIssueForm(t);
  const input = await openAssigneePicker();
  await act(() => pressEscape(input));

  assert.equal(environment.container.querySelector('[data-testid="modal-open"]')?.textContent, "true");
  assert.equal(
    document.querySelector('input[placeholder="search"]'),
    null,
    "the picker should consume Escape and close"
  );
});

await test("normal work-item form Escape still closes the form", async (t) => {
  const environment = await renderIssueForm(t);
  const title = environment.container.querySelector('input[name="name"]');
  assert.ok(title);
  await act(() => pressEscape(title));

  assert.equal(environment.container.querySelector('[data-testid="modal-open"]')?.textContent, "false");
});

await test("work-item form Escape stays blocked while the editor is not ready", async (t) => {
  const environment = await renderIssueForm(t);
  state.editorReady = false;
  t.after(() => {
    state.editorReady = true;
  });
  const title = environment.container.querySelector('input[name="name"]');
  assert.ok(title);
  await act(() => pressEscape(title));

  assert.equal(environment.container.querySelector('[data-testid="modal-open"]')?.textContent, "true");
});

await test("work-item form Escape stays blocked while saving", async (t) => {
  let resolveSave;
  const save = new Promise((resolve) => {
    resolveSave = resolve;
  });
  const environment = await renderIssueForm(t, { onSubmit: () => save });
  const form = environment.container.querySelector("form");
  assert.ok(form);
  await act(async () => form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  assert.equal(environment.container.querySelector("fieldset")?.disabled, true);
  await act(() => pressEscape(form));

  assert.equal(environment.container.querySelector('[data-testid="modal-open"]')?.textContent, "true");
  await act(async () => resolveSave());
});

await test("work-item form Escape stays blocked while read-only", async (t) => {
  const environment = await renderIssueForm(t, { readOnly: true });
  const form = environment.container.querySelector("form");
  assert.ok(form);
  await act(() => pressEscape(form));

  assert.equal(environment.container.querySelector('[data-testid="modal-open"]')?.textContent, "true");
});

await test("work-item form Escape stays blocked while the assistant is open", async (t) => {
  const environment = await renderIssueForm(t, {}, (env) =>
    env.client.setQueryData(["instance", "info"], {
      instance: { is_setup_done: true },
      config: { has_llm_configured: true },
    })
  );
  const openAssistant = [...document.querySelectorAll("button")].find((button) =>
    button.textContent.includes("open-assistant")
  );
  assert.ok(openAssistant, "the assistant trigger should be available when the instance has AI configured");
  await act(async () => openAssistant.click());
  const form = environment.container.querySelector("form");
  assert.ok(form);
  await act(() => pressEscape(form));

  assert.equal(environment.container.querySelector('[data-testid="modal-open"]')?.textContent, "true");
});

await test("intake form keeps its modal open when a portaled assignee picker consumes Escape", async (t) => {
  const environment = await setup(t);
  globalThis.issueFormCreatePortal = webRequire("react-dom").createPortal;
  const { InboxIssueCreateRoot } =
    await import("../../apps/web/core/components/inbox/modals/create-modal/create-root.tsx");
  state.intake = { create: async () => ({ issue: { id: "created" } }) };

  await environment.render(
    React.createElement(ModalHarness, null, (handleModalClose) =>
      React.createElement(InboxIssueCreateRoot, {
        workspaceSlug: "workspace",
        projectId: "first",
        handleModalClose,
      })
    )
  );
  const input = await openAssigneePicker();
  await act(() => pressEscape(input));

  assert.equal(environment.container.querySelector('[data-testid="modal-open"]')?.textContent, "true");
  assert.equal(document.querySelector('input[placeholder="search"]'), null);
});

await test("scoped Escape shortcuts ignore prevented, unrelated, and portaled events", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.unmount();
  });
  const { default: useKeypress } = await import("../../apps/web/core/hooks/use-keypress.tsx");
  const createPortal = webRequire("react-dom").createPortal;
  let calls = 0;

  function Harness() {
    const targetRef = React.useRef(null);
    useKeypress("Escape", () => calls++, targetRef);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement(
        "form",
        { ref: targetRef },
        React.createElement("input", { "data-testid": "inside" }),
        React.createElement("input", {
          "data-testid": "handled",
          onKeyDown: (event) => event.preventDefault(),
        })
      ),
      createPortal(React.createElement("input", { "data-testid": "portal" }), document.body),
      React.createElement("input", { "data-testid": "outside" })
    );
  }

  await environment.render(React.createElement(Harness));
  await act(() => pressEscape(environment.container.querySelector('[data-testid="outside"]')));
  assert.equal(calls, 0, "events outside the form should not invoke its shortcut");
  await act(() => pressEscape(environment.container.querySelector('[data-testid="handled"]')));
  assert.equal(calls, 0, "a child-consumed Escape should not invoke the shortcut");
  await act(() => pressEscape(document.querySelector('[data-testid="portal"]')));
  assert.equal(calls, 0, "a portaled picker should not invoke the form shortcut");
  await act(() => pressEscape(environment.container.querySelector('[data-testid="inside"]')));
  assert.equal(calls, 1, "an unconsumed Escape inside the form should invoke the shortcut once");
});

import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const observed = [];
const packageStubs = {
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/hooks": "export const useOutsideClickDetector = () => {};",
  "@plane/propel/icons":
    "export const ChevronDownIcon = () => null; export const ChevronRightIcon = () => null; export const MoreHorizontal = () => null; export const PlusIcon = () => null;",
  "@plane/propel/toast": "export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = () => {};",
  "@plane/types": "export const EIssueServiceType = { ISSUES: 'issues' };",
  "@/providers/issue-detail-ui-state": `export const useIssueDetailUiState = () => ({ toggleIssueLinkModal: value => globalThis.__widgetCalls.push(['link', value]), setLastWidgetAction: value => globalThis.__widgetCalls.push(['last', value]) });`,
  "@/hooks/use-file-size": "export const useFileSize = () => ({ maxFileSize: 1024 });",
  "react-dropzone": `export const useDropzone = ({ disabled }) => ({ getRootProps: (props = {}) => ({ ...props, onClick: () => { if (!disabled) globalThis.__widgetCalls.push(['dropzone-click']); }, onKeyDown: () => {}, role: props.role ?? 'presentation', tabIndex: disabled ? undefined : props.tabIndex ?? 0 }), getInputProps: () => ({ type: 'file', disabled }) });`,
  "@plane/propel/tooltip": "export const Tooltip = ({ children }) => children;",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "./helper" && context.parentURL?.includes("/issue-detail-widgets/attachments/")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const useAttachmentOperations = () => ({ operations: { create: async () => {} } });")}`,
      };
    }
    if (specifier === "../utils" && context.parentURL?.includes("/packages/ui/src/dropdowns/")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');")}`,
      };
    }
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    if (specifier === "react-popper") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });")}`,
      };
    }
    return nextResolve(specifier, context);
  },
});

globalThis.__widgetCalls = observed;

let components;
async function loadComponents() {
  components ??= Promise.all([
    import("../../packages/ui/src/dropdowns/custom-menu.tsx"),
    import("../../apps/web/core/components/issues/issue-detail-widgets/widget-button.tsx"),
    import("../../apps/web/core/components/issues/issue-detail-widgets/links/quick-action-button.tsx"),
    import("../../apps/web/core/components/issues/issue-detail-widgets/attachments/quick-action-button.tsx"),
  ]).then(([menu, button, links, attachments]) => ({
    CustomMenu: menu.CustomMenu,
    IssueDetailWidgetButton: button.IssueDetailWidgetButton,
    IssueLinksActionButton: links.IssueLinksActionButton,
    IssueAttachmentActionButton: attachments.IssueAttachmentActionButton,
  }));
  return components;
}

test("issue detail widget button is a ref-forwarding single CustomMenu trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const consoleErrors = [];
  const originalConsoleError = console.error;
  console.error = (...args) => consoleErrors.push(args.join(" "));
  t.after(() => {
    console.error = originalConsoleError;
  });
  const { CustomMenu, IssueDetailWidgetButton } = await loadComponents();
  const ref = React.createRef();
  await environment.render(
    React.createElement(
      CustomMenu,
      {
        customButton: React.createElement(IssueDetailWidgetButton, { ref, title: "Add sub-work item" }),
        customButtonAsChild: true,
      },
      React.createElement(CustomMenu.MenuItem, null, "Create new")
    )
  );

  const trigger = environment.container.querySelector("button");
  assert.equal(environment.container.querySelectorAll("button").length, 1);
  assert.equal(ref.current, trigger);
  assert.equal(trigger.textContent, "Add sub-work item");
  assert.equal(
    consoleErrors.some((message) => message.includes("Function components cannot be given refs")),
    false
  );
  await React.act(async () => trigger.click());
  assert.ok(document.body.textContent.includes("Create new"));
});

test("link and attachment actions use their supplied buttons as the only interactive trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { IssueDetailWidgetButton, IssueLinksActionButton, IssueAttachmentActionButton } = await loadComponents();
  observed.length = 0;
  await environment.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(IssueLinksActionButton, {
        customButton: React.createElement(IssueDetailWidgetButton, {
          title: "Add link",
          onClick: () => observed.push(["child-click"]),
        }),
        issueServiceType: "issues",
      }),
      React.createElement(IssueAttachmentActionButton, {
        attachmentHelpers: { operations: { create: async () => {} } },
        workspaceSlug: "default",
        projectId: "project-id",
        issueId: "issue-id",
        customButton: React.createElement(IssueDetailWidgetButton, { title: "Attach" }),
        issueServiceType: "issues",
      }),
      React.createElement(IssueLinksActionButton, {
        customButton: React.createElement(IssueDetailWidgetButton, { title: "Disabled link" }),
        disabled: true,
        issueServiceType: "issues",
      }),
      React.createElement(IssueAttachmentActionButton, {
        attachmentHelpers: { operations: { create: async () => {} } },
        workspaceSlug: "default",
        projectId: "project-id",
        issueId: "issue-id",
        customButton: React.createElement(IssueDetailWidgetButton, { title: "Disabled attachment" }),
        disabled: true,
        issueServiceType: "issues",
      }),
      React.createElement(IssueLinksActionButton, {
        customButton: React.createElement("span", null, "Link icon"),
        issueServiceType: "issues",
      }),
      React.createElement(IssueAttachmentActionButton, {
        attachmentHelpers: { operations: { create: async () => {} } },
        workspaceSlug: "default",
        projectId: "project-id",
        issueId: "issue-id",
        customButton: React.createElement("span", null, "Attachment icon"),
        issueServiceType: "issues",
      }),
      React.createElement(IssueAttachmentActionButton, {
        attachmentHelpers: { operations: { create: async () => {} } },
        workspaceSlug: "default",
        projectId: "project-id",
        issueId: "issue-id",
        customButton: React.createElement(IssueDetailWidgetButton, { title: "Child disabled", disabled: true }),
        issueServiceType: "issues",
      })
    )
  );

  const buttons = [...environment.container.querySelectorAll("button")];
  assert.equal(buttons.length, 7);
  assert.equal(
    buttons.some((button) => button.querySelector("button")),
    false
  );
  assert.deepEqual(
    buttons.map((button) => button.disabled),
    [false, false, true, true, false, false, true]
  );
  assert.equal(environment.container.querySelectorAll('input[type="file"]').length, 4);
  assert.equal(environment.container.querySelectorAll('input[type="file"]')[1].disabled, true);
  assert.equal(environment.container.querySelectorAll('input[type="file"]')[3].disabled, true);
  const dropzoneRoots = [...environment.container.querySelectorAll('div[role="presentation"]')];
  assert.equal(dropzoneRoots.length, 4);
  assert.equal(dropzoneRoots[0].getAttribute("tabindex"), "-1");
  assert.equal(dropzoneRoots[3].hasAttribute("tabindex"), false);
  assert.equal(environment.container.querySelectorAll('[tabindex="0"], button:not([disabled])').length, 4);
  assert.equal(buttons[4].textContent, "Link icon");
  assert.equal(buttons[5].textContent, "Attachment icon");
  await React.act(async () => buttons[0].click());
  assert.deepEqual(observed, [["child-click"], ["link", true]]);
  await React.act(async () => buttons[1].click());
  assert.deepEqual(observed, [["child-click"], ["link", true], ["dropzone-click"]]);
  await React.act(async () => buttons[2].click());
  await React.act(async () => buttons[3].click());
  assert.deepEqual(observed, [["child-click"], ["link", true], ["dropzone-click"]]);
  await React.act(async () => buttons[4].click());
  await React.act(async () => buttons[5].click());
  await React.act(async () => buttons[6].click());
  await React.act(async () => dropzoneRoots[3].click());
  assert.deepEqual(observed, [["child-click"], ["link", true], ["dropzone-click"], ["link", true], ["dropzone-click"]]);
});

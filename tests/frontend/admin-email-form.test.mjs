import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import {
  setupAdminSettingsFormTest,
  adminRequire,
  changeInput,
  formButton,
  formToasts,
  settleForm,
  submitAndWait,
} from "./helpers/admin-forms.mjs";
const emailSource = "apps/admin/app/(all)/(dashboard)/email/email-config-form.tsx";
const testEmailSource = "apps/admin/app/(all)/(dashboard)/email/test-email-modal.tsx";
const generalSource = "apps/admin/app/(all)/(dashboard)/general/form.tsx";
const nativeForms = setupAdminSettingsFormTest([emailSource, testEmailSource, generalSource]);
const { InstanceService } = adminRequire("@plane/services");
const config = {
  EMAIL_HOST: "smtp.example.com",
  EMAIL_PORT: "587",
  EMAIL_FROM: "from@example.com",
  EMAIL_HOST_USER: "",
  EMAIL_HOST_PASSWORD: "",
  EMAIL_USE_TLS: "1",
  EMAIL_USE_SSL: "0",
  ENABLE_SMTP: "1",
};

async function mount(t, Component, props, existingEnvironment) {
  const environment = existingEnvironment ?? (await createReactDomTestEnvironment());
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  const render = (values) =>
    environment.render(React.createElement(QueryClientProvider, { client }, React.createElement(Component, values)));
  t.after(async () => {
    await environment.unmount();
    client.clear();
  });
  await render(props);
  return {
    ...environment,
    renderProps: render,
    button: (label) => formButton(document.body, label),
    queryClient: client,
  };
}

function waitForText(root, selector, pattern) {
  const findMatch = () => {
    const element = root.querySelector(selector);
    return element && pattern.test(element.textContent) ? element : null;
  };
  const immediateMatch = findMatch();
  if (immediateMatch) return Promise.resolve(immediateMatch);
  return new Promise((resolve, reject) => {
    const observer = new MutationObserver(() => {
      const match = findMatch();
      if (!match) return;
      clearTimeout(timeout);
      observer.disconnect();
      resolve(match);
    });
    observer.observe(root, { childList: true, characterData: true, subtree: true });
    const timeout = setTimeout(() => {
      observer.disconnect();
      reject(new Error(`Timed out waiting for ${selector} matching ${pattern}`));
    }, 3000);
  });
}

await test("security-only edits save exclusive TLS/SSL flags, failures retain the draft, success resets save state", async (t) => {
  const dom = await createReactDomTestEnvironment();
  const { InstanceEmailForm } = await import("../../apps/admin/app/(all)/(dashboard)/email/email-config-form.tsx");
  let fail = true;
  const payloads = [];
  t.mock.method(InstanceService.prototype, "updateConfigurations", async (payload) => {
    payloads.push(payload);
    if (fail) throw new Error("offline");
    return Object.entries(payload).map(([key, value]) => ({ key, value }));
  });
  const view = await mount(t, InstanceEmailForm, { config }, dom);
  assert.equal(view.button("Save changes").disabled, true);
  assert.equal(view.button("Send test email").disabled, false);
  const security = view.container.querySelector("select");
  await act(async () => {
    security.value = "ssl";
    security.dispatchEvent(new Event("change", { bubbles: true }));
  });
  assert.equal(view.button("Save changes").disabled, false);
  assert.equal(view.button("Send test email").disabled, true);
  await view.renderProps({ config: { ...config, EMAIL_HOST: "server-refresh.example.com" } });
  assert.equal(
    view.container.querySelector('[name="EMAIL_HOST"]').value,
    "smtp.example.com",
    "refetch must not overwrite draft"
  );
  const formApi = nativeForms.getForms(emailSource).at(-1);
  await submitAndWait(view.button("Save changes"), formApi);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(formToasts.at(-1).type, "error");
  assert.equal(security.value, "ssl");
  assert.equal(view.button("Save changes").disabled, false);
  fail = false;
  await submitAndWait(view.button("Save changes"), formApi);
  assert.equal(formApi.state.isSubmitSuccessful, true);
  assert.equal(view.button("Save changes").disabled, true);
  assert.equal(view.button("Send test email").disabled, false);
  assert.deepEqual(payloads[1], { ...config, EMAIL_USE_TLS: "0", EMAIL_USE_SSL: "1" });
});

await test("invalid SMTP ports do not save", async (t) => {
  const { InstanceEmailForm } = await import("../../apps/admin/app/(all)/(dashboard)/email/email-config-form.tsx");
  const save = t.mock.method(InstanceService.prototype, "updateConfigurations", async () => []);
  const view = await mount(t, InstanceEmailForm, { config });
  await changeInput(view.container.querySelector('[name="EMAIL_PORT"]'), "70000");
  await act(async () => view.button("Save changes").click());
  await settleForm();
  assert.equal(save.mock.callCount(), 0);
  assert.match(view.container.querySelector('[role="alert"]').textContent, /port/);
});

await test("test email validates recipient and lets a failed request retry without losing it", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { SendTestEmailModal } = await import("../../apps/admin/app/(all)/(dashboard)/email/test-email-modal.tsx");
  let fail = true;
  const recipients = [];
  t.mock.method(InstanceService.prototype, "sendTestEmail", async (recipient) => {
    recipients.push(recipient);
    if (fail) throw new Error("offline");
    return {};
  });
  const view = await mount(t, SendTestEmailModal, { handleClose() {} }, environment);
  // Headless UI portals the dialog outside the root container returned by the test environment.
  const input = document.body.querySelector('[name="receiverEmail"]');
  await changeInput(input, "not-an-address");
  assert.equal(view.button("Send email").disabled, true);
  assert.match(document.body.querySelector('[role="alert"]').textContent, /valid email address/i);
  assert.equal(recipients.length, 0);
  await changeInput(input, "recipient@example.com");
  const formApi = nativeForms.getForms(testEmailSource).at(-1);
  await submitAndWait(view.button("Send email"), formApi);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(input.value, "recipient@example.com");
  const failedMutation = view.queryClient.getMutationCache().getAll().at(-1);
  assert.equal(failedMutation.state.status, "error");
  assert.ok(failedMutation.state.error);
  const sendError = await act(async () =>
    waitForText(document.body, '[role="alert"]', /failed to send the test email/i)
  );
  assert.match(sendError.textContent, /failed to send the test email/i);
  assert.equal(view.button("Send email").disabled, false, "the retained recipient remains available for retry");
  fail = false;
  await submitAndWait(view.button("Send email"), formApi);
  assert.equal(formApi.state.isSubmitSuccessful, true);
  assert.deepEqual(recipients, ["recipient@example.com", "recipient@example.com"]);
  assert.equal(view.queryClient.getMutationCache().getAll().at(-1).state.status, "success");
  await act(async () => waitForText(document.body, "h2", /Email sent/));
});

await test("saved SMTP credentials can be enabled without changing another field", async (t) => {
  const { InstanceEmailForm } = await import("../../apps/admin/app/(all)/(dashboard)/email/email-config-form.tsx");
  const save = t.mock.method(InstanceService.prototype, "updateConfigurations", async (payload) =>
    Object.entries(payload).map(([key, value]) => ({ key, value }))
  );
  const view = await mount(t, InstanceEmailForm, { config: { ...config, ENABLE_SMTP: "0" } });
  assert.equal(view.button("Save changes").disabled, false);
  assert.equal(view.button("Send test email").disabled, true);
  await act(async () => view.button("Save changes").click());
  await settleForm();
  assert.equal(save.mock.calls[0].arguments[0].ENABLE_SMTP, "1");
});

await test("instance settings retain edits after a failed save and submit the same draft on retry", async (t) => {
  const { GeneralConfigurationForm } = await import("../../apps/admin/app/(all)/(dashboard)/general/form.tsx");
  const original = { instance_name: "Vector", is_telemetry_enabled: false, instance_id: "instance-1" };
  t.mock.method(InstanceService.prototype, "telemetry", async () => ({
    logs_and_traces: { status: "enabled", destination: "http://logal:4318" },
    instance_metrics: { status: "not_configured", destination: null },
    analytics_events: { status: "not_configured", destination: null },
  }));
  let fail = true;
  const payloads = [];
  t.mock.method(InstanceService.prototype, "update", async (payload) => {
    payloads.push(payload);
    if (fail) throw new Error("offline");
    return { ...original, ...payload };
  });
  const view = await mount(t, GeneralConfigurationForm, {
    instance: original,
    instanceAdmins: [{ user_detail: { email: "admin@example.com" } }],
  });
  const name = view.container.querySelector('[name="instance_name"]');
  await changeInput(name, "Vector Test");
  assert.equal(view.container.querySelector('[role="switch"]'), null);
  const formApi = nativeForms.getForms(generalSource).at(-1);
  await submitAndWait(view.button("Save changes"), formApi);
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.equal(name.value, "Vector Test");
  assert.equal(view.button("Save changes").disabled, false);
  assert.equal(formToasts.at(-1).type, "error");
  fail = false;
  await submitAndWait(view.button("Save changes"), formApi);
  assert.equal(formApi.state.isSubmitSuccessful, true);
  assert.deepEqual(payloads, [{ instance_name: "Vector Test" }, { instance_name: "Vector Test" }]);
  assert.equal(view.button("Save changes").disabled, true);
});

await test("SMTP cannot be toggled while a configuration save is pending", async (t) => {
  const { default: EmailPage } = await import("../../apps/admin/app/(all)/(dashboard)/email/page.tsx");
  const { instanceConfigurationsOptions } = await import("../../apps/admin/queries/instance.ts");
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient();
  t.after(async () => {
    await environment.unmount();
    client.clear();
  });
  const saved = Object.entries(config).map(([key, value]) => ({ key, value }));
  client.setQueryData(
    instanceConfigurationsOptions.queryKey,
    saved.map((item) => (item.key === "ENABLE_SMTP" ? { ...item, value: "0" } : item))
  );
  let completeSave;
  t.mock.method(
    InstanceService.prototype,
    "updateConfigurations",
    () =>
      new Promise((resolve) => {
        completeSave = () => resolve(saved);
      })
  );
  t.mock.method(InstanceService.prototype, "configurations", async () => saved);
  await environment.render(React.createElement(QueryClientProvider, { client }, React.createElement(EmailPage)));
  const toggle = environment.container.querySelector('[role="switch"]');
  await act(async () => toggle.click());
  await act(async () => formButton(environment.container, "Save changes").click());
  await settleForm();
  assert.equal(toggle.disabled, true);
  await act(async () => {
    completeSave();
  });
  await settleForm();
  assert.equal(toggle.disabled, false);
  assert.equal(toggle.getAttribute("aria-checked"), "true");
});

await test("telemetry shows configured destinations and hides stale status on failure until retry succeeds", async (t) => {
  const { TelemetryStatus } = await import("../../apps/admin/app/(all)/(dashboard)/general/telemetry-status.tsx");
  const { instanceTelemetryOptions } = await import("../../apps/admin/queries/instance.ts");
  let fail = false;
  t.mock.method(InstanceService.prototype, "telemetry", async () => {
    if (fail) throw new Error("offline");
    return {
      logs_and_traces: { status: "enabled", destination: "http://logal:4318" },
      instance_metrics: { status: "disabled", destination: "https://metrics.example.test" },
      analytics_events: { status: "not_configured", destination: null },
    };
  });
  const view = await mount(t, TelemetryStatus, {});
  await settleForm();
  assert.match(view.container.textContent, /Enabled.*Destination: http:\/\/logal:4318/);
  assert.match(view.container.textContent, /Disabled.*Configured destination: https:\/\/metrics.example.test/);
  assert.match(view.container.textContent, /Not configured.*No destination configured/);
  fail = true;
  await act(async () => view.queryClient.refetchQueries({ queryKey: instanceTelemetryOptions.queryKey }));
  await settleForm();
  assert.match(view.container.querySelector('[role="alert"]').textContent, /Unable to load/);
  assert.doesNotMatch(view.container.textContent, /logal:4318/);
  fail = false;
  await act(async () => view.button("Retry").click());
  await settleForm();
  assert.match(view.container.textContent, /logal:4318/);
  assert.equal(view.container.querySelector('[role="alert"]'), null);
});

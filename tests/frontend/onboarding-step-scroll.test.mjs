import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
registerWebTestModules(
  {
    "./profile": "export const ProfileSetupStep=()=>null;",
    "./role": "export const RoleSetupStep=()=>null;",
    "./team": "export const InviteTeamStep=()=>null;",
    "./usecase": "export const UseCaseSetupStep=()=>null;",
    "./workspace": "export const WorkspaceSetupStep=()=>null;",
  },
  (specifier, context, next) => {
    // Step fixtures must not replace similarly named workspace-package exports.
    const scoped = specifier.startsWith(".") && context.parentURL?.includes("/packages/");
    return next(scoped ? new URL(specifier, context.parentURL).href : specifier, context);
  }
);

await test("onboarding resets scroll when a step mounts, retaining scroll during same-step updates", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const calls = [];
  const elementPrototype = HTMLElement.prototype;
  Object.defineProperty(elementPrototype, "scrollTo", {
    configurable: true,
    value(options) {
      calls.push({ element: this, options });
    },
  });
  t.after(() => {
    delete elementPrototype.scrollTo;
  });
  const { OnboardingStepRoot } = await import("../../apps/web/core/components/onboarding/steps/root.tsx");
  const { EOnboardingSteps } = await import("../../packages/types/dist/index.mjs");
  const render = (currentStep) =>
    env.render(
      React.createElement(OnboardingStepRoot, {
        currentStep,
        invitations: [],
        handleStepChange: async () => {},
      })
    );
  await render(EOnboardingSteps.PROFILE_SETUP);
  assert.equal(calls.length, 1);
  assert.equal(calls[0].element, env.container.firstElementChild);
  assert.deepEqual(calls[0].options, { top: 0, behavior: "smooth" });
  await render(EOnboardingSteps.PROFILE_SETUP);
  assert.equal(calls.length, 1);
  await render(EOnboardingSteps.ROLE_SETUP);
  assert.equal(calls.length, 2);
});

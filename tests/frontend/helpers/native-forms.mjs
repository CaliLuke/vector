import { act } from "./react.mjs";

export async function changeFormControl(input, value) {
  await act(async () => {
    const prototype =
      input.tagName === "SELECT"
        ? HTMLSelectElement.prototype
        : input.tagName === "TEXTAREA"
          ? HTMLTextAreaElement.prototype
          : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(prototype, "value").set.call(input, value);
    input.dispatchEvent(new Event(input.tagName === "SELECT" ? "change" : "input", { bubbles: true }));
  });
}
export const submitForm = (container) =>
  act(async () => {
    container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });

export function waitForSubmitCompletion(form) {
  if (!form.state.isSubmitting) return Promise.resolve();
  let timeout;
  const completed = new Promise((resolve) => {
    const subscription = form.store.subscribe(() => {
      if (!form.state.isSubmitting) {
        clearTimeout(timeout);
        subscription.unsubscribe();
        resolve();
      }
    });
  });
  return Promise.race([
    completed,
    new Promise((_, reject) => {
      timeout = setTimeout(() => reject(new Error("native form submission did not settle")), 3000);
    }),
  ]).finally(() => clearTimeout(timeout));
}

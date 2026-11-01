import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, test } from "vitest";
import { AuthCodeResend } from "./code-resend";

test("resend controls retain their hint and block clicks while delivery or cooldown is active", () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  let attempts = 0;
  const resend = () => {
    attempts += 1;
  };
  try {
    act(() =>
      root.render(
        <AuthCodeResend hint="Check your email" disabled onResend={resend}>
          Resend in 5s
        </AuthCodeResend>
      )
    );
    const button = container.querySelector("button")!;
    expect(button.type).toBe("button");
    expect(button.disabled).toBe(true);
    act(() => button.click());
    expect(attempts).toBe(0);
    act(() =>
      root.render(
        <AuthCodeResend hint="Check your email" disabled={false} onResend={resend}>
          Resend
        </AuthCodeResend>
      )
    );
    expect(container.textContent).toBe("Check your emailResend");
    expect(button.disabled).toBe(false);
    act(() => button.click());
    expect(attempts).toBe(1);
  } finally {
    act(() => root.unmount());
  }
});

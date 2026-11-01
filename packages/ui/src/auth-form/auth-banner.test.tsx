import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { AuthMessageBanner } from "./auth-banner";

test("authentication messages dismiss without submitting their surrounding form", () => {
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  const dismiss = vi.fn(),
    submit = vi.fn();
  try {
    act(() =>
      root.render(
        <form onSubmit={submit}>
          <AuthMessageBanner bannerData={{ message: "Code expired" }} handleBannerData={dismiss} />
        </form>
      )
    );
    expect(host.querySelector('[role="alert"]')?.textContent).toContain("Code expired");
    const button = host.querySelector("button");
    expect(button?.type).toBe("button");
    act(() => button?.click());
    expect(dismiss).toHaveBeenCalledExactlyOnceWith(undefined);
    expect(submit).not.toHaveBeenCalled();
    act(() => root.render(<AuthMessageBanner bannerData={{ message: "Sign-in unavailable" }} />));
    expect(host.textContent).toBe("Sign-in unavailable");
    expect(host.querySelector("button")).toBeNull();
    act(() => root.render(<AuthMessageBanner bannerData={undefined} />));
    expect(host.textContent).toBe("");
  } finally {
    act(() => root.unmount());
    host.remove();
  }
});

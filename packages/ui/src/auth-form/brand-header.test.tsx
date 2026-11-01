import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, test, vi } from "vitest";
import { BrandHeader } from "./brand-header";

test("branding preserves the application's native home-link behavior", () => {
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  const navigate = vi.fn((event: React.MouseEvent<HTMLAnchorElement>) => event.preventDefault());
  try {
    act(() =>
      root.render(
        <BrandHeader
          homeLink={<a href="/administration/" aria-label="Administration home" onClick={navigate} data-home />}
        />
      )
    );
    const link = host.querySelector("a");
    expect(link?.getAttribute("href")).toBe("/administration/");
    expect(link?.hasAttribute("data-home")).toBe(true);
    expect(link?.getAttribute("aria-label")).toBe("Administration home");
    expect(host.querySelectorAll("a")).toHaveLength(1);
    expect(link?.querySelector("svg")?.getAttribute("height")).toBe("20");
    expect(link?.querySelector("svg")?.getAttribute("width")).toBe("95");
    act(() => link?.click());
    expect(navigate).toHaveBeenCalledOnce();
  } finally {
    act(() => root.unmount());
    host.remove();
  }
});

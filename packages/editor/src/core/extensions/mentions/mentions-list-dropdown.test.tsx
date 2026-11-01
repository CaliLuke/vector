import { act, createRef } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import type { TMentionSection } from "@/types";
import { MentionsListDropdown } from "./mentions-list-dropdown";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
let host: HTMLDivElement;
let root: Root;
beforeEach(() => {
  vi.useFakeTimers();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
  vi.useRealTimers();
});

function suggestions(title: string): TMentionSection[] {
  return [
    { key: "users", items: [{ title, id: title, entity_identifier: title, entity_name: "user_mention", icon: null }] },
  ];
}
function deferred() {
  let resolve!: (value: TMentionSection[]) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<TMentionSection[]>((accept, fail) => {
    resolve = accept;
    reject = fail;
  });
  return { promise, resolve, reject };
}

test("only the current query can publish suggestions or stop its loading state", async () => {
  const old = deferred();
  const current = deferred();
  const searchCallback = vi.fn((query: string) => (query === "old" ? old.promise : current.promise));
  const command = vi.fn();
  const onClose = vi.fn();
  const ref = createRef<{ onKeyDown: (props: { event: KeyboardEvent }) => boolean }>();
  act(() =>
    root.render(
      <MentionsListDropdown ref={ref} query="old" command={command} searchCallback={searchCallback} onClose={onClose} />
    )
  );
  await act(() => vi.advanceTimersByTimeAsync(300));
  act(() =>
    root.render(
      <MentionsListDropdown
        ref={ref}
        query="current"
        command={command}
        searchCallback={searchCallback}
        onClose={onClose}
      />
    )
  );
  await act(async () => {
    old.resolve(suggestions("Obsolete"));
  });
  expect(host.textContent).toBe("Loading...");
  act(() => {
    ref.current?.onKeyDown({ event: new KeyboardEvent("keydown", { key: "Enter" }) });
  });
  expect(command).not.toHaveBeenCalled();
  await act(() => vi.advanceTimersByTimeAsync(300));
  await act(async () => {
    current.resolve(suggestions("Current"));
  });
  expect(host.textContent).toBe("Current");
  act(() => host.querySelector("button")?.click());
  expect(command).toHaveBeenCalledWith(expect.objectContaining({ title: "Current", entity_identifier: "Current" }));
});

test("a late older response cannot overwrite current results", async () => {
  const old = deferred();
  const searchCallback = vi.fn((query: string) =>
    query === "old" ? old.promise : Promise.resolve(suggestions("Current"))
  );
  const command = vi.fn();
  const onClose = vi.fn();
  act(() =>
    root.render(
      <MentionsListDropdown query="old" command={command} searchCallback={searchCallback} onClose={onClose} />
    )
  );
  await act(() => vi.advanceTimersByTimeAsync(300));
  act(() =>
    root.render(
      <MentionsListDropdown query="current" command={command} searchCallback={searchCallback} onClose={onClose} />
    )
  );
  await act(() => vi.advanceTimersByTimeAsync(300));
  expect(host.textContent).toBe("Current");
  await act(async () => {
    old.resolve(suggestions("Obsolete"));
  });
  expect(host.textContent).toBe("Current");
});

test("query replacement and unmount cancel searches that have not started", async () => {
  const searchCallback = vi.fn(async () => suggestions("Result"));
  const command = vi.fn();
  const onClose = vi.fn();
  act(() =>
    root.render(
      <MentionsListDropdown query="first" command={command} searchCallback={searchCallback} onClose={onClose} />
    )
  );
  await act(() => vi.advanceTimersByTimeAsync(200));
  act(() =>
    root.render(
      <MentionsListDropdown query="second" command={command} searchCallback={searchCallback} onClose={onClose} />
    )
  );
  await act(() => vi.advanceTimersByTimeAsync(200));
  expect(searchCallback).not.toHaveBeenCalled();
  act(() => root.render(null));
  await act(() => vi.advanceTimersByTimeAsync(300));
  expect(searchCallback).not.toHaveBeenCalled();
});

test("a failed current search clears loading and cannot retain earlier suggestions", async () => {
  const failure = deferred();
  const searchCallback = vi.fn((query: string) =>
    query === "first" ? Promise.resolve(suggestions("First")) : failure.promise
  );
  const command = vi.fn();
  const onClose = vi.fn();
  const error = vi.spyOn(console, "error").mockImplementation(() => {});
  try {
    act(() =>
      root.render(
        <MentionsListDropdown query="first" command={command} searchCallback={searchCallback} onClose={onClose} />
      )
    );
    await act(() => vi.advanceTimersByTimeAsync(300));
    expect(host.textContent).toBe("First");
    act(() =>
      root.render(
        <MentionsListDropdown query="failed" command={command} searchCallback={searchCallback} onClose={onClose} />
      )
    );
    expect(host.querySelector("button")).toBeNull();
    await act(() => vi.advanceTimersByTimeAsync(300));
    await act(async () => {
      failure.reject(new Error("Search unavailable"));
    });
    expect(host.textContent).toBe("No results");
  } finally {
    error.mockRestore();
  }
});

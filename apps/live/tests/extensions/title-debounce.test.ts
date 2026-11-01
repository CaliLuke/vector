import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DebounceManager } from "@/extensions/title-update/debounce";

vi.mock("@plane/logger", () => ({ logger: { error: vi.fn() } }));

describe("title debounce", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it("coalesces edits into the latest title after the full quiet period", async () => {
    const manager = new DebounceManager<[string]>({ wait: 50 });
    const save = vi.fn(async () => {});
    manager.schedule(save, "First");
    await vi.advanceTimersByTimeAsync(30);
    manager.schedule(save, "Latest");
    await vi.advanceTimersByTimeAsync(49);
    expect(save).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);
    expect(save).toHaveBeenCalledExactlyOnceWith("Latest", expect.any(AbortSignal));
    await manager.flush(save);
    expect(save).toHaveBeenCalledOnce();
  });

  it("flushes a pending title once and cancels its delayed execution", async () => {
    const manager = new DebounceManager<[string]>({ wait: 50 });
    const save = vi.fn(async () => {});
    manager.schedule(save, "Pending");
    await manager.flush(save);
    await vi.advanceTimersByTimeAsync(100);
    expect(save).toHaveBeenCalledExactlyOnceWith("Pending", expect.any(AbortSignal));
  });

  it("retries a rejected save with the same title", async () => {
    const manager = new DebounceManager<[string]>({ wait: 50 });
    const save = vi
      .fn<(...args: [string, AbortSignal]) => Promise<void>>()
      .mockRejectedValueOnce(new Error("Offline"))
      .mockResolvedValue(undefined);
    manager.schedule(save, "Unsaved");
    await vi.advanceTimersByTimeAsync(50);
    expect(save).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(50);
    expect(save).toHaveBeenLastCalledWith("Unsaved", expect.any(AbortSignal));
    expect(save).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(100);
    expect(save).toHaveBeenCalledTimes(2);
  });

  it("cancels pending edits and aborts an active save without replaying it", async () => {
    const manager = new DebounceManager<[string]>({ wait: 50 });
    let finishSave: (() => void) | undefined;
    const completion = new Promise<void>((resolve) => {
      finishSave = resolve;
    });
    const signals: AbortSignal[] = [];
    const save = vi.fn((_title: string, signal: AbortSignal) => {
      signals.push(signal);
      return completion;
    });
    manager.schedule(save, "Pending");
    manager.cancel();
    await vi.advanceTimersByTimeAsync(100);
    expect(save).not.toHaveBeenCalled();
    manager.schedule(save, "Active");
    await vi.advanceTimersByTimeAsync(50);
    manager.cancel();
    expect(signals[0]?.aborted).toBe(true);
    finishSave?.();
    await vi.advanceTimersByTimeAsync(100);
    await manager.flush(save);
    expect(save).toHaveBeenCalledTimes(1);
  });

  it("aborts an obsolete active call before saving a newer title", async () => {
    const manager = new DebounceManager<[string]>({ wait: 50 });
    const signals: AbortSignal[] = [];
    const save = vi.fn((title: string, signal: AbortSignal) => {
      signals.push(signal);
      if (title !== "Old") return Promise.resolve();
      return new Promise<void>((_resolve, reject) => {
        signal.addEventListener("abort", () => reject(new DOMException("Cancelled", "AbortError")), { once: true });
      });
    });
    manager.schedule(save, "Old");
    await vi.advanceTimersByTimeAsync(50);
    manager.schedule(save, "New");
    await vi.advanceTimersByTimeAsync(50);
    expect(signals[0]?.aborted).toBe(true);
    await vi.advanceTimersByTimeAsync(20);
    expect(save).toHaveBeenLastCalledWith("New", expect.any(AbortSignal));
    expect(signals[1]?.aborted).toBe(false);
    await vi.advanceTimersByTimeAsync(100);
    expect(save).toHaveBeenCalledTimes(2);
  });
});

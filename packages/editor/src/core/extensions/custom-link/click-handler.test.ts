import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { afterEach, expect, test, vi } from "vitest";
import { clickHandler } from "./helpers/clickHandler";

const editors: Editor[] = [];
afterEach(() => {
  for (const editor of editors.splice(0)) editor.destroy();
  vi.restoreAllMocks();
});
function fixture() {
  const editor = new Editor({ extensions: [StarterKit.configure({ link: false })], content: "<p>Link</p>" });
  editors.push(editor);
  const anchor = document.createElement("a");
  anchor.href = "https://example.com/actual";
  anchor.target = "_blank";
  anchor.innerHTML = "<span>Link text</span><svg><rect /></svg>";
  editor.view.dom.querySelector("p")!.append(anchor);
  const plugin = clickHandler();
  const handle = plugin.props.handleClick;
  if (!handle) throw new Error("missing click handler");
  return {
    anchor,
    click(this: void, target: Node, button = 0) {
      const event = new MouseEvent("click", { button, cancelable: true });
      event.preventDefault();
      target.dispatchEvent(event);
      return handle.call(plugin, editor.view, 1, event);
    },
  };
}

test("editor link clicks use the clicked anchor URL and target for nested text and SVG content", () => {
  const open = vi.spyOn(window, "open").mockReturnValue(null);
  const { anchor, click } = fixture();
  expect(click(anchor.querySelector("span")!.firstChild!)).toBe(true);
  expect(click(anchor.querySelector("rect")!)).toBe(true);
  expect(open).toHaveBeenCalledTimes(2);
  expect(open).toHaveBeenLastCalledWith("https://example.com/actual", "_blank");
});

test.each(["javascript:alert(1)", "data:text/html,unsafe", "file:///tmp/file", "about:blank", "vbscript:unsafe"])(
  "editor link clicks reject %s even when it reaches the DOM",
  (href) => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    const { anchor, click } = fixture();
    anchor.setAttribute("href", href);
    expect(click(anchor)).toBe(false);
    expect(open).not.toHaveBeenCalled();
  }
);

test("non-primary clicks, missing hrefs, non-link nodes and targets outside the editor are not opened", () => {
  const open = vi.spyOn(window, "open").mockReturnValue(null);
  const { anchor, click } = fixture();
  expect(click(anchor, 1)).toBe(false);
  expect(click(anchor.parentElement!)).toBe(false);
  const outside = document.createElement("a");
  outside.href = "https://example.com/outside";
  document.body.append(outside);
  expect(click(outside)).toBe(false);
  outside.remove();
  anchor.removeAttribute("href");
  expect(click(anchor)).toBe(false);
  expect(open).not.toHaveBeenCalled();
});

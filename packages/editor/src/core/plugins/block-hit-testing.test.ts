import { afterEach, expect, test } from "vitest";
import { nodeDOMAtCoords } from "./block-hit-testing";

const originalHitTest = Object.getOwnPropertyDescriptor(document, "elementsFromPoint");

afterEach(() => {
  document.body.replaceChildren();
  if (originalHitTest) Object.defineProperty(document, "elementsFromPoint", originalHitTest);
  else Reflect.deleteProperty(document, "elementsFromPoint");
});

function hit(markup: string, selectors: string[]) {
  document.body.innerHTML = markup;
  const elements = selectors.map((selector) => {
    const element = document.querySelector(selector);
    if (!element) throw new Error(`Missing hit target ${selector}`);
    return element;
  });
  // jsdom has no layout engine; the browser hit stack is the boundary under test.
  Object.defineProperty(document, "elementsFromPoint", { configurable: true, value: () => elements });
  return nodeDOMAtCoords({ x: 30, y: 40 });
}

test("a table cell resolves to its table while drag previews are ignored", () => {
  expect(
    hit('<table id="table"><tbody><tr><td id="cell"><p id="text">Text</p></td></tr></tbody></table>', [
      "#text",
      "#cell",
      "#table",
    ])?.id
  ).toBe("table");
  expect(
    hit('<table class="table-drag-preview"><tbody><tr><td>Preview</td></tr></tbody></table>', ["td", "table"])
  ).toBeNull();
});

test("embedded content resolves to its owning block rather than an inner paragraph", () => {
  expect(
    hit('<div class="editor-embed-component" id="embed"><p class="editor-paragraph-block" id="inner">Text</p></div>', [
      "#inner",
      "#embed",
    ])?.id
  ).toBe("embed");
});

test("first editor paragraphs and ordinary eligible blocks are selectable", () => {
  expect(
    hit(
      '<div class="ProseMirror"><p id="first">First</p><p class="editor-paragraph-block" id="second">Second</p></div>',
      ["#first"]
    )?.id
  ).toBe("first");
  expect(
    hit('<div class="ProseMirror"><p>First</p><p class="editor-paragraph-block" id="second">Second</p></div>', [
      "#second",
    ])?.id
  ).toBe("second");
});

test("unrecognized nodes and empty hit stacks do not select a block", () => {
  expect(hit('<div><p id="plain">Outside</p><span id="text">Text</span></div>', ["#text", "#plain"])).toBeNull();
  expect(hit("", [])).toBeNull();
});

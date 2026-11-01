import { Editor } from "@tiptap/core";
import { EditorContent } from "@tiptap/react";
import { StarterKit } from "@tiptap/starter-kit";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, expect, test, vi } from "vitest";
import { createEditorHookProps } from "../../../tests/helpers/editor-hook-props";
import { CustomMentionExtension } from "./mentions/extension";
import { CustomCalloutExtension } from "./callout/extension";
import { DEFAULT_CALLOUT_BLOCK_ATTRIBUTES, getStoredLogo, updateStoredLogo } from "./callout/utils";
import { getImageAttributes } from "./custom-image/utils";
import { ECustomImageStatus } from "./custom-image/types";
import { UtilityExtension } from "./utility";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
afterEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
});

test.each([null, [], 4, { in_use: "emoji", emoji: { value: {} } }, { in_use: "icon", icon: { name: 4 } }])(
  "invalid stored callout logo %j falls back without interrupting insertion",
  (value) => {
    localStorage.setItem("editor-calloutComponent-logo", JSON.stringify(value));
    const editor = new Editor({ extensions: [StarterKit, CustomCalloutExtension], content: "<p>Text</p>" });
    try {
      expect(editor.commands.insertCallout()).toBe(true);
      expect(editor.getJSON().content?.[0]).toMatchObject({
        type: "calloutComponent",
        attrs: {
          "data-logo-in-use": "emoji",
          "data-emoji-unicode": DEFAULT_CALLOUT_BLOCK_ATTRIBUTES["data-emoji-unicode"],
        },
      });
    } finally {
      editor.destroy();
    }
  }
);

test("stored callout icons and emoji retain valid fields and reject invalid optional fields", () => {
  updateStoredLogo({ in_use: "icon", icon: { name: "Star", color: "#123456" } });
  expect(getStoredLogo()).toEqual({
    "data-logo-in-use": "icon",
    "data-icon-name": "Star",
    "data-icon-color": "#123456",
  });
  updateStoredLogo({ in_use: "emoji", emoji: { value: "128512", url: "https://example.com/emoji.png" } });
  expect(getStoredLogo()).toEqual({
    "data-logo-in-use": "emoji",
    "data-emoji-unicode": "128512",
    "data-emoji-url": "https://example.com/emoji.png",
  });
  localStorage.setItem(
    "editor-calloutComponent-logo",
    JSON.stringify({ in_use: "emoji", emoji: { value: "128512", url: {} } })
  );
  expect(getStoredLogo()).toMatchObject({ "data-emoji-url": DEFAULT_CALLOUT_BLOCK_ATTRIBUTES["data-emoji-url"] });
  localStorage.setItem("editor-calloutComponent-logo", "malformed JSON");
  vi.spyOn(console, "error").mockImplementation(() => {});
  expect(getStoredLogo()).toMatchObject({ "data-logo-in-use": "emoji" });
  expect(localStorage.getItem("editor-calloutComponent-logo")).toBeNull();
});

test("imported mentions validate entity fields before invoking UI and text handlers", async () => {
  const renderMention = vi.fn(() => <span>Mention</span>);
  const getDetails = vi.fn(() => ({ display_name: "Someone" }));
  const editor = new Editor({
    extensions: [
      StarterKit,
      CustomMentionExtension(),
      UtilityExtension({
        ...createEditorHookProps({
          mentionHandler: {
            searchCallback: async () => [],
            renderComponent: renderMention,
            getMentionedEntityDetails: getDetails,
          },
        }),
        isEditable: true,
        isTouchDevice: false,
      }),
    ],
    content: {
      type: "doc",
      content: [
        {
          type: "paragraph",
          content: [{ type: "mention", attrs: { id: {}, entity_identifier: 42, entity_name: "unknown" } }],
        },
      ],
    },
  });
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  try {
    await act(async () => root.render(<EditorContent editor={editor} />));
    expect(host.textContent).toBe("Mention");
    expect(renderMention).toHaveBeenLastCalledWith({ entity_identifier: "", entity_name: "user_mention" });
    expect(editor.getText()).toBe("@Someone");
    expect(getDetails).toHaveBeenLastCalledWith("");
    await act(async () => {
      editor.commands.setContent({
        type: "doc",
        content: [
          {
            type: "paragraph",
            content: [{ type: "mention", attrs: { id: "item", entity_identifier: "item-id", entity_name: "issue" } }],
          },
        ],
      });
    });
    expect(renderMention).toHaveBeenLastCalledWith({ entity_identifier: "item-id", entity_name: "issue" });
  } finally {
    await act(async () => root.unmount());
    editor.destroy();
    host.remove();
  }
});

test("image dimensions accept existing numeric and pixel values and bound malformed imports", () => {
  expect(
    getImageAttributes({
      id: "image",
      src: "asset",
      width: 400,
      height: "200px",
      aspectRatio: 2,
      alignment: "right",
      status: "uploaded",
    })
  ).toEqual({
    id: "image",
    src: "asset",
    width: 400,
    height: "200px",
    aspectRatio: 2,
    alignment: "right",
    status: ECustomImageStatus.UPLOADED,
  });
  expect(
    getImageAttributes({
      id: {},
      src: 42,
      width: "invalid",
      height: -3,
      aspectRatio: Infinity,
      alignment: "invalid",
      status: "invalid",
    })
  ).toEqual({
    id: null,
    src: null,
    width: "35%",
    height: "auto",
    aspectRatio: null,
    alignment: "left",
    status: ECustomImageStatus.PENDING,
  });
});

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEditorState, useEditor as useTiptapEditor } from "@tiptap/react";
import { useImperativeHandle, useEffect, useLayoutEffect, useMemo } from "react";
import type { EditorState } from "@tiptap/pm/state";
import type { MarkdownStorage } from "tiptap-markdown";
// extensions
import { CoreEditorExtensions } from "@/extensions";
// helpers
import { getEditorRefHelpers } from "@/helpers/editor-ref";
// props
import { CoreEditorProps } from "@/props";
// types
import type { TEditorHookProps } from "@/types";

declare module "@tiptap/core" {
  interface Storage {
    markdown: MarkdownStorage;
  }
}

const emptyEditorProps: NonNullable<TEditorHookProps["editorProps"]> = {};

export const useEditor = (props: TEditorHookProps) => {
  const {
    autofocus = false,
    disabledExtensions,
    editable = true,
    editorClassName = "",
    editorProps = emptyEditorProps,
    enableHistory,
    extendedEditorProps,
    extensions = [],
    fileHandler,
    flaggedExtensions,
    forwardedRef,
    getEditorMetaData,
    handleEditorReady,
    id = "",
    initialValue,
    isTouchDevice,
    mentionHandler,
    onAssetChange,
    onChange,
    onEditorFocus,
    onTransaction,
    placeholder,
    showPlaceholderOnEmpty,
    tabIndex,
    provider,
    value,
  } = props;

  const currentEditorProps = useMemo(() => {
    const defaults = CoreEditorProps({ editorClassName });
    return {
      ...defaults,
      ...editorProps,
      attributes: (state: EditorState) => ({
        ...(typeof defaults.attributes === "function" ? defaults.attributes(state) : defaults.attributes),
        ...(typeof editorProps.attributes === "function" ? editorProps.attributes(state) : editorProps.attributes),
      }),
    };
  }, [editorClassName, editorProps]);
  const editor = useTiptapEditor(
    {
      editable,
      immediatelyRender: false,
      shouldRerenderOnTransaction: false,
      autofocus,
      parseOptions: { preserveWhitespace: true },
      editorProps: currentEditorProps,
      extensions: [
        ...CoreEditorExtensions({
          disabledExtensions,
          editable,
          enableHistory,
          extendedEditorProps,
          fileHandler,
          flaggedExtensions,
          getEditorMetaData,
          isTouchDevice,
          mentionHandler,
          placeholder,
          showPlaceholderOnEmpty,
          tabIndex,
          provider,
        }),
        ...extensions,
      ],
      content: initialValue,
      onCreate: () => handleEditorReady?.(true),
      onTransaction: () => {
        onTransaction?.();
      },
      onUpdate: ({ editor, transaction }) => {
        // Check if this update is only due to migration update
        const isMigrationUpdate = transaction?.getMeta("uniqueIdOnlyChange") === true;
        onChange?.(editor.getJSON(), editor.getHTML(), { isMigrationUpdate });
      },
      onDestroy: () => handleEditorReady?.(false),
      onFocus: onEditorFocus,
    },
    [editable]
  );

  // Refresh field labels/errors without recreating the editor or its undo history.
  useLayoutEffect(() => {
    if (editor && !editor.isDestroyed) editor.setOptions({ editorProps: currentEditorProps });
  }, [editor, currentEditorProps]);

  // Extensions and plugins outlive React renders. Keep their identity callbacks
  // in editor storage and update them only after the new props commit.
  useLayoutEffect(() => {
    if (editor && !editor.isDestroyed) {
      editor.commands.updateIdentityHandlers({ getEditorMetaData, mentionHandler });
    }
  }, [editor, getEditorMetaData, mentionHandler]);

  useLayoutEffect(() => {
    if (editor && !editor.isDestroyed) editor.commands.updateFileHandler(fileHandler);
  }, [editor, fileHandler]);

  // Effect for syncing query data
  useEffect(() => {
    // value is null when intentionally passed where syncing is not yet
    // supported and value is undefined when the query data is not populated
    if (value == null) return;
    if (editor && !editor.isDestroyed) {
      const { uploadInProgress: isUploadInProgress } = editor.storage.utility;
      if (!editor.isDestroyed && !isUploadInProgress) {
        try {
          editor.commands.setContent(value, {
            emitUpdate: false,
            parseOptions: { preserveWhitespace: true },
          });
          if (editor.state.selection) {
            const docLength = editor.state.doc.content.size;
            const relativePosition = Math.min(editor.state.selection.from, docLength - 1);
            editor.commands.setTextSelection(relativePosition);
          }
        } catch (error) {
          console.error("Error syncing editor content with external value:", error);
        }
      }
    }
  }, [editor, value, id]);

  // update assets upload status
  useEffect(() => {
    if (!editor || editor.isDestroyed) return;
    const assetsUploadStatus = fileHandler.assetsUploadStatus;
    editor.commands.updateAssetsUploadStatus?.(assetsUploadStatus);
  }, [editor, fileHandler.assetsUploadStatus]);

  // subscribe to assets list changes
  const assetsList = useEditorState({
    editor,
    selector: ({ editor }) => ({
      assets: editor?.storage.utility?.assetsList ?? [],
    }),
  });
  // trigger callback when assets list changes
  useEffect(() => {
    const assets = assetsList?.assets;
    if (!assets || !onAssetChange) return;
    onAssetChange(assets);
  }, [assetsList?.assets, onAssetChange]);

  useImperativeHandle(
    forwardedRef,
    () =>
      getEditorRefHelpers({
        editor,
        getEditorMetaData,
        provider,
      }),
    [editor, getEditorMetaData, provider]
  );

  if (!editor) {
    return null;
  }

  return editor;
};

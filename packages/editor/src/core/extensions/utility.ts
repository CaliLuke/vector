/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Extension } from "@tiptap/core";
import codemark from "prosemirror-codemark";
// helpers
import { CORE_EXTENSIONS } from "@/constants/extension";
import { restorePublicImages } from "@/helpers/image-helpers";
// plugins
import type { TAdditionalActiveDropbarExtensions } from "@/plane-editor/types/utils";
import { DropHandlerPlugin } from "@/plugins/drop";
import { FilePlugins } from "@/plugins/file/root";
import { MarkdownClipboardPlugin } from "@/plugins/markdown-clipboard";
import type { IEditorProps, TEditorAsset, TFileHandler, TMentionHandler } from "@/types";

type TEditorIdentityHandlers = {
  getEditorMetaData: IEditorProps["getEditorMetaData"];
  mentionHandler: TMentionHandler;
};

type TActiveDropbarExtensions =
  | CORE_EXTENSIONS.MENTION
  | CORE_EXTENSIONS.EMOJI
  | CORE_EXTENSIONS.SLASH_COMMANDS
  | CORE_EXTENSIONS.TABLE
  | "bubble-menu"
  | CORE_EXTENSIONS.SIDE_MENU
  | TAdditionalActiveDropbarExtensions;

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    [CORE_EXTENSIONS.UTILITY]: {
      updateFileHandler: (fileHandler: TFileHandler) => ReturnType;
      updateIdentityHandlers: (handlers: TEditorIdentityHandlers) => ReturnType;
      setUploadActive: (id: symbol, active: boolean) => ReturnType;
      updateAssetsUploadStatus: (updatedStatus: TFileHandler["assetsUploadStatus"]) => () => void;
      updateAssetsList: (
        args:
          | {
              asset: TEditorAsset;
            }
          | {
              idToRemove: string;
            }
      ) => () => void;
      addActiveDropbarExtension: (extension: TActiveDropbarExtensions) => () => void;
      removeActiveDropbarExtension: (extension: TActiveDropbarExtensions) => () => void;
    };
  }
  interface Storage {
    [CORE_EXTENSIONS.UTILITY]: UtilityExtensionStorage;
  }
}

export type UtilityExtensionStorage = {
  fileHandler: TFileHandler;
  identityHandlers: TEditorIdentityHandlers;
  assetsList: TEditorAsset[];
  assetsUploadStatus: TFileHandler["assetsUploadStatus"];
  activeUploads: Set<symbol>;
  uploadInProgress: boolean;
  activeDropbarExtensions: TActiveDropbarExtensions[];
  isTouchDevice: boolean;
};

type Props = Pick<IEditorProps, "disabledExtensions" | "flaggedExtensions" | "getEditorMetaData"> & {
  fileHandler: TFileHandler;
  isEditable: boolean;
  isTouchDevice: boolean;
  mentionHandler: TMentionHandler;
};

export const UtilityExtension = (props: Props) => {
  const {
    disabledExtensions,
    flaggedExtensions,
    fileHandler,
    getEditorMetaData,
    isEditable,
    isTouchDevice,
    mentionHandler,
  } = props;

  return Extension.create<Record<string, unknown>, UtilityExtensionStorage>({
    name: CORE_EXTENSIONS.UTILITY,
    priority: 1000,

    addProseMirrorPlugins() {
      return [
        ...FilePlugins({
          editor: this.editor,
          isEditable,
          fileHandler,
        }),
        ...codemark({ markType: this.editor.schema.marks.code }),
        MarkdownClipboardPlugin({
          editor: this.editor,
          getEditorMetaData: (htmlContent) => this.storage.identityHandlers.getEditorMetaData(htmlContent),
        }),
        DropHandlerPlugin({
          disabledExtensions,
          flaggedExtensions,
          editor: this.editor,
        }),
      ];
    },

    onCreate() {
      restorePublicImages(this.editor, (src) => this.storage.fileHandler.restore(src));
    },

    addStorage() {
      return {
        fileHandler,
        identityHandlers: { getEditorMetaData, mentionHandler },
        assetsList: [],
        assetsUploadStatus: isEditable && "assetsUploadStatus" in fileHandler ? fileHandler.assetsUploadStatus : {},
        activeUploads: new Set<symbol>(),
        uploadInProgress: false,
        activeDropbarExtensions: [],
        isTouchDevice,
      };
    },

    addCommands() {
      return {
        updateFileHandler: (nextFileHandler) => () => {
          this.storage.fileHandler = nextFileHandler;
          return true;
        },
        updateIdentityHandlers:
          (nextHandlers) =>
          ({ tr }) => {
            this.storage.identityHandlers = nextHandlers;
            tr.setMeta("preventDispatch", true);
            return true;
          },
        setUploadActive: (id, active) => () => {
          if (active) this.storage.activeUploads.add(id);
          else this.storage.activeUploads.delete(id);
          this.storage.uploadInProgress = this.storage.activeUploads.size > 0;
          return true;
        },
        updateAssetsUploadStatus: (updatedStatus) => () => {
          this.storage.assetsUploadStatus = updatedStatus;
        },
        updateAssetsList: (args) => () => {
          const uniqueAssets = new Set(this.storage.assetsList);
          if ("asset" in args) {
            const alreadyExists = this.storage.assetsList.find((asset) => asset.id === args.asset.id);
            if (!alreadyExists) {
              uniqueAssets.add(args.asset);
            }
          } else if ("idToRemove" in args) {
            const asset = this.storage.assetsList.find((asset) => asset.id === args.idToRemove);
            if (asset) {
              uniqueAssets.delete(asset);
            }
          }
          this.storage.assetsList = Array.from(uniqueAssets);
        },
        addActiveDropbarExtension: (extension) => () => {
          const index = this.storage.activeDropbarExtensions.indexOf(extension);
          if (index === -1) {
            this.storage.activeDropbarExtensions.push(extension);
          }
        },
        removeActiveDropbarExtension: (extension) => () => {
          const index = this.storage.activeDropbarExtensions.indexOf(extension);
          if (index !== -1) {
            this.storage.activeDropbarExtensions.splice(index, 1);
          }
        },
      };
    },
  });
};

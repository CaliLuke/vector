/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPage } from "@plane/types";

import { useCallback, useMemo, useRef } from "react";
// plane imports
import { LIVE_BASE_PATH, LIVE_BASE_URL } from "@plane/constants";
import { CollaborativeDocumentEditorWithRef } from "@plane/editor";
import type {
  CollaborationState,
  EditorRefApi,
  EditorTitleRefApi,
  TAIMenuProps,
  TDisplayConfig,
  TFileHandler,
  TRealtimeConfig,
  TServerHandler,
} from "@plane/editor";
import { useTranslation } from "@plane/i18n";
import type { TSearchEntityRequestPayload, TSearchResponse, TWebhookConnectionQueryParams } from "@plane/types";
import { ERowVariant, Row } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { generateRandomColor, hslToHex } from "@plane/utils/color";
// components
import { EditorMentionsRoot } from "@/components/editor/embeds/mentions";
// hooks
import { useEditorMention } from "@/hooks/editor";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { usePageRecord } from "@/components/pages/page-record-context";
import { usePageFilters } from "@/hooks/use-page-filters";
import { useCurrentUser } from "@/hooks/use-current-user";
import { usePagePermissions } from "@/hooks/use-page-permissions";
import { useParseEditorContent } from "@/hooks/use-parse-editor-content";
// plane web imports
import type { TCustomEventHandlers } from "@/hooks/use-realtime-page-events";
import { useRealtimePageEvents } from "@/hooks/use-realtime-page-events";
import type { TExtendedEditorExtensionsConfig } from "@/hooks/pages";
import { DISABLED_EDITOR_EXTENSIONS } from "@/components/editor/extensions";
// store
// local imports
import { PageContentLoader } from "../loaders/page-content-loader";
import { PageEditorHeaderRoot } from "./header";
import { PageContentBrowser } from "./summary";
import { EditorAIMenu } from "./ai/menu";

export type TEditorBodyConfig = {
  fileHandler: TFileHandler;
};

export type TEditorBodyHandlers = {
  fetchEntity: (payload: TSearchEntityRequestPayload) => Promise<TSearchResponse>;
  getRedirectionLink: (pageId?: string) => string;
};

type Props = {
  config: TEditorBodyConfig;
  editorForwardRef: React.RefObject<EditorRefApi | null>;
  handleEditorReady: (status: boolean) => void;
  handleOpenNavigationPane: () => void;
  handlers: TEditorBodyHandlers;
  isNavigationPaneOpen: boolean;
  page: TPage;
  webhookConnectionParams: TWebhookConnectionQueryParams;
  projectId?: string;
  workspaceSlug: string;
  customRealtimeEventHandlers?: TCustomEventHandlers;
  extendedEditorProps: TExtendedEditorExtensionsConfig;
  isFetchingFallbackBinary?: boolean;
  onCollaborationStateChange?: (state: CollaborationState) => void;
};

export function PageEditorBody(props: Props) {
  const {
    config,
    editorForwardRef,
    handleEditorReady,
    handleOpenNavigationPane,
    handlers,
    isNavigationPaneOpen,
    page,
    webhookConnectionParams,
    projectId,
    workspaceSlug,
    extendedEditorProps,
    isFetchingFallbackBinary,
    onCollaborationStateChange,
  } = props;
  // refs
  const titleEditorRef = useRef<EditorTitleRefApi>(null);
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { isContentEditable } = usePagePermissions(page, workspaceSlug);
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const getMemberDisplayName = useCallback(
    (userId: string) => membersById?.get(userId)?.member.display_name,
    [membersById]
  );
  // derived values
  const { id: pageId } = page;
  const { editorRef, setAssets: updateAssetsList, setSyncStatus: setSyncingStatus } = usePageRecord();
  const workspaceId = workspaceDetails?.id ?? "";
  // use editor mention
  const { fetchMentions } = useEditorMention({
    enableAdvancedMentions: true,
    searchEntity: handlers.fetchEntity,
  });
  // parse content
  const { getEditorMetaData } = useParseEditorContent({
    projectId,
    workspaceSlug,
  });
  // page filters
  const { fontSize, fontStyle, isFullWidth } = usePageFilters();
  // translation
  const { t } = useTranslation();
  // derived values
  const displayConfig: TDisplayConfig = useMemo(
    () => ({
      fontSize,
      fontStyle,
      wideLayout: isFullWidth,
    }),
    [fontSize, fontStyle, isFullWidth]
  );

  // Use the new hook to handle page events
  const { updatePageProperties } = useRealtimePageEvents({
    page,
    getMemberDisplayName,
    handlers,
  });

  const getAIMenu = useCallback(
    ({ isOpen, onClose }: TAIMenuProps) => (
      <EditorAIMenu
        editorRef={editorRef}
        isOpen={isOpen}
        onClose={onClose}
        workspaceId={workspaceId}
        workspaceSlug={workspaceSlug}
      />
    ),
    [editorRef, workspaceId, workspaceSlug]
  );

  const serverHandler: TServerHandler = useMemo(
    () => ({
      onStateChange: (state) => {
        // Pass full state to parent
        onCollaborationStateChange?.(state);

        // Map collaboration stage to UI syncing status
        // Stage → UI mapping: disconnected → error | synced → synced | all others → syncing
        if (state.stage.kind === "disconnected") {
          setSyncingStatus("error");
        } else if (state.stage.kind === "synced") {
          setSyncingStatus("synced");
        } else {
          // initial, connecting, awaiting-sync, reconnecting → show as syncing
          setSyncingStatus("syncing");
        }
      },
    }),
    [setSyncingStatus, onCollaborationStateChange]
  );

  const realtimeConfig: TRealtimeConfig | undefined = useMemo(() => {
    // The collaboration URL is derived from window.location, so it can only be
    // built on the client. There is no socket to connect to during SSR anyway —
    // the config is recomputed on hydration.
    if (typeof window === "undefined") return undefined;

    // Construct the WebSocket Collaboration URL
    try {
      const LIVE_SERVER_BASE_URL = LIVE_BASE_URL?.trim() || window.location.origin;
      const WS_LIVE_URL = new URL(LIVE_SERVER_BASE_URL);
      const isSecureEnvironment = window.location.protocol === "https:";
      WS_LIVE_URL.protocol = isSecureEnvironment ? "wss" : "ws";
      WS_LIVE_URL.pathname = `${LIVE_BASE_PATH}/collaboration`;

      // Append query parameters to the URL
      Object.entries(webhookConnectionParams)
        .filter(([_, value]) => value !== undefined && value !== null)
        .forEach(([key, value]) => {
          WS_LIVE_URL.searchParams.set(key, value);
        });

      // Construct realtime config
      return {
        url: WS_LIVE_URL.toString(),
        documentName: page.collaboration_revision ? `${pageId}.${page.collaboration_revision}` : pageId,
      };
    } catch (error) {
      console.error("Error creating realtime config", error);
      return undefined;
    }
  }, [webhookConnectionParams, pageId, page.collaboration_revision]);

  const userConfig = useMemo(
    () => ({
      id: currentUser?.id ?? "",
      name: currentUser?.display_name ?? "",
      color: hslToHex(generateRandomColor(currentUser?.id ?? "")),
    }),
    [currentUser?.display_name, currentUser?.id]
  );

  const blockWidthClassName = cn(
    "mx-auto block w-full max-w-[720px] bg-transparent transition-all duration-200 ease-in-out",
    {
      "max-w-[1152px]": isFullWidth,
    }
  );

  const isPageLoading = pageId === undefined || !realtimeConfig;

  if (isPageLoading) return <PageContentLoader className={blockWidthClassName} />;

  return (
    <Row
      className="vertical-scrollbar relative flex scrollbar-md size-full flex-col overflow-x-hidden overflow-y-auto duration-200"
      variant={ERowVariant.HUGGING}
    >
      <div id="page-content-container" className="relative w-full flex-shrink-0">
        {/* table of content */}
        {!isNavigationPaneOpen && (
          <div className="page-summary-container absolute top-[64px] right-0 z-[5] h-full">
            <div className="sticky top-[72px]">
              <div className="group/page-toc relative px-page-x">
                <button
                  type="button"
                  className="max-h-[50vh] !cursor-pointer overflow-hidden"
                  aria-label={t("page_navigation_pane.outline_floating_button")}
                  onClick={handleOpenNavigationPane}
                >
                  <PageContentBrowser className="overflow-y-auto" editorRef={editorRef} showOutline />
                </button>
                <div className="vertical-scrollbar pointer-events-none absolute top-0 right-0 scrollbar-sm max-h-[70vh] w-52 translate-x-1/2 overflow-y-scroll rounded-sm bg-surface-2 p-4 whitespace-nowrap opacity-0 transition-all duration-300 group-hover/page-toc:pointer-events-auto group-hover/page-toc:-translate-x-1/4 group-hover/page-toc:opacity-100">
                  <PageContentBrowser className="overflow-y-auto" editorRef={editorRef} />
                </div>
              </div>
            </div>
          </div>
        )}
        <div>
          <div className="page-header-container group/page-header">
            <div className={blockWidthClassName}>
              <PageEditorHeaderRoot page={page} projectId={projectId} workspaceSlug={workspaceSlug} />
            </div>
          </div>
          <CollaborativeDocumentEditorWithRef
            editable={isContentEditable}
            id={pageId}
            fileHandler={config.fileHandler}
            handleEditorReady={handleEditorReady}
            ref={editorForwardRef}
            titleRef={titleEditorRef}
            containerClassName="h-full p-0 pb-64"
            displayConfig={displayConfig}
            getEditorMetaData={getEditorMetaData}
            mentionHandler={{
              searchCallback: async (query) => {
                const res = await fetchMentions(query);
                if (!res) throw new Error("Failed in fetching mentions");
                return res;
              },
              // oxlint-disable-next-line no-shadow
              renderComponent: (props) => <EditorMentionsRoot {...props} />,
              getMentionedEntityDetails: (id: string) => {
                const member = membersById?.get(id)?.member;
                return member ? { display_name: member.display_name } : undefined;
              },
            }}
            updatePageProperties={updatePageProperties}
            realtimeConfig={realtimeConfig}
            serverHandler={serverHandler}
            user={userConfig}
            disabledExtensions={DISABLED_EDITOR_EXTENSIONS}
            flaggedExtensions={[]}
            aiHandler={{
              menu: getAIMenu,
            }}
            onAssetChange={updateAssetsList}
            extendedEditorProps={extendedEditorProps}
            isFetchingFallbackBinary={isFetchingFallbackBinary}
          />
        </div>
      </div>
    </Row>
  );
}

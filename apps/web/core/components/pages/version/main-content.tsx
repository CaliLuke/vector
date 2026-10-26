/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { usePageRecord } from "@/components/pages/page-record-context";
import { projectPageVersionOptions } from "@/queries/pages";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { EyeIcon, TriangleAlert } from "lucide-react";
// plane imports
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { renderFormattedDate, renderFormattedTime } from "@plane/utils/datetime";
// helpers
// local imports
import type { TVersionEditorProps } from "./editor";

type Props = {
  activeVersion: string | null;
  editorComponent: React.FC<TVersionEditorProps>;
  handleClose: () => void;
  handleRestore: (descriptionHTML: string) => Promise<void>;
  pageId: string;
  restoreEnabled: boolean;
};

export function PageVersionsMainContent(props: Props) {
  const { activeVersion, editorComponent, handleClose, handleRestore, pageId, restoreEnabled } = props;
  const { scope } = usePageRecord();
  const client = useQueryClient();
  // states
  const [isRestoring, setIsRestoring] = useState(false);

  const {
    data: versionDetails,
    error: versionDetailsError,
    refetch,
    isFetching: isRetrying,
  } = useQuery(projectPageVersionOptions(client, scope, pageId, activeVersion));

  const handleRestoreVersion = async () => {
    if (!restoreEnabled || !versionDetails) return;
    setIsRestoring(true);
    await handleRestore(versionDetails?.description_html ?? "<p></p>")
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Page version restored.",
        });
        handleClose();
      })
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Failed to restore page version.",
        })
      )
      .finally(() => setIsRestoring(false));
  };

  const VersionEditor = editorComponent;

  return (
    <div className="flex flex-grow flex-col overflow-hidden">
      {versionDetailsError ? (
        <div className="grid flex-grow place-items-center">
          <div className="flex flex-col items-center gap-4 text-center">
            <span className="grid size-11 flex-shrink-0 place-items-center text-tertiary">
              <TriangleAlert className="size-10" />
            </span>
            <div>
              <h6 className="text-16 font-semibold">Something went wrong!</h6>
              <p className="text-13 text-tertiary">The version could not be loaded, please try again.</p>
            </div>
            <Button variant="link" onClick={() => void refetch()} loading={isRetrying}>
              Try again
            </Button>
          </div>
        </div>
      ) : (
        <>
          <div className="flex min-h-14 items-center justify-between gap-2 border-b border-subtle px-5 py-3">
            <div className="flex items-center gap-4">
              <h6 className="text-14 font-medium">
                {versionDetails
                  ? `${renderFormattedDate(versionDetails.last_saved_at)} ${renderFormattedTime(versionDetails.last_saved_at)}`
                  : "Loading version details"}
              </h6>
              <span className="flex flex-shrink-0 items-center gap-1 rounded-sm bg-accent-primary/20 px-1.5 py-1 text-11 font-medium text-accent-primary">
                <EyeIcon className="size-3 flex-shrink-0" />
                View only
              </span>
            </div>
            {restoreEnabled && (
              <Button
                variant="primary"
                className="flex-shrink-0"
                onClick={handleRestoreVersion}
                loading={isRestoring}
                disabled={!versionDetails}
              >
                {isRestoring ? "Restoring" : "Restore"}
              </Button>
            )}
          </div>
          <div className="vertical-scrollbar scrollbar-sm h-full overflow-y-scroll pt-8">
            <VersionEditor activeVersion={activeVersion} versionDetails={versionDetails} />
          </div>
        </>
      )}
    </div>
  );
}

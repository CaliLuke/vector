/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment, useEffect, useRef } from "react";
import { observer } from "mobx-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Dialog, Transition } from "@headlessui/react";
// hooks
import { useIssueDetails } from "@/hooks/store/use-issue-details";
// local imports
import { FullScreenPeekView } from "./full-screen-peek-view";
import { SidePeekView } from "./side-peek-view";

type TIssuePeekOverview = {
  anchor: string;
  peekId: string;
  handlePeekClose?: () => void;
};

export const IssuePeekOverview = observer(function IssuePeekOverview(props: TIssuePeekOverview) {
  const { anchor, peekId, handlePeekClose } = props;
  const router = useRouter();
  const searchParams = useSearchParams();
  // query params
  const board = searchParams.get("board") || undefined;
  const state = searchParams.get("state") || undefined;
  const priority = searchParams.get("priority") || undefined;
  const labels = searchParams.get("labels") || undefined;
  // store
  const { peekMode, setPeekId, getIssueById, fetchIssueDetails } = useIssueDetails();
  const panelRef = useRef<HTMLDivElement>(null);
  const previousPeekModeRef = useRef(peekMode);
  // derived values
  const issueDetails = peekId ? getIssueById(peekId.toString()) : undefined;
  // state
  const isPeekOpen = !!peekId;
  const isSidePeek = peekMode === "side";
  const isModalPeekOpen = !!peekId && (peekMode === "modal" || peekMode === "full");

  useEffect(() => {
    if (anchor && peekId) {
      fetchIssueDetails(anchor, peekId.toString());
    }
  }, [anchor, fetchIssueDetails, peekId]);

  useEffect(() => {
    if (previousPeekModeRef.current === peekMode) return;
    previousPeekModeRef.current = peekMode;
    // The full-screen view replaces the mode selector, so keep focus inside the persistent dialog.
    const panel = panelRef.current;
    panel?.focus();
  }, [peekMode]);

  const handleClose = () => {
    // if close logic is passed down, call that instead of the below logic
    if (handlePeekClose) {
      handlePeekClose();
      return;
    }

    setPeekId(null);
    let queryParams: any = {
      board,
    };
    if (priority && priority.length > 0) queryParams = { ...queryParams, priority: priority };
    if (state && state.length > 0) queryParams = { ...queryParams, state: state };
    if (labels && labels.length > 0) queryParams = { ...queryParams, labels: labels };
    queryParams = new URLSearchParams(queryParams).toString();
    router.push(`/issues/${anchor}?${queryParams}`);
  };

  return (
    <>
      <Transition.Root appear show={isPeekOpen} as={Fragment}>
        <Dialog as="div" onClose={handleClose}>
          <Transition
            as={Fragment}
            show={isModalPeekOpen}
            enter="ease-out duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="ease-in duration-200"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div className="fixed inset-0 z-20 bg-backdrop transition-opacity" />
          </Transition>
          <Transition.Child
            as={Dialog.Panel}
            className="pointer-events-none fixed inset-0 z-20"
            enter="transition-opacity duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="transition-opacity duration-200"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div
              ref={panelRef}
              tabIndex={-1}
              className={
                isSidePeek
                  ? "pointer-events-auto fixed top-0 right-0 z-20 h-full w-1/2 border-l border-subtle-1 bg-surface-1 shadow-raised-200 transition-all duration-300"
                  : `pointer-events-auto fixed top-1/2 left-1/2 z-20 -translate-x-1/2 -translate-y-1/2 rounded-lg bg-surface-1 transition-all duration-300 ${
                      peekMode === "modal" ? "h-[70%] w-3/5" : "size-[95%]"
                    }`
              }
            >
              {peekMode === "full" ? (
                <FullScreenPeekView anchor={anchor} handleClose={handleClose} issueDetails={issueDetails} />
              ) : (
                <SidePeekView anchor={anchor} handleClose={handleClose} issueDetails={issueDetails} />
              )}
            </div>
          </Transition.Child>
        </Dialog>
      </Transition.Root>
    </>
  );
});

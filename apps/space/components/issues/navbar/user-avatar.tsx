/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useForm, useStore } from "@tanstack/react-form";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { currentUserOptions } from "@/queries/session";
import { useTheme } from "next-themes";
import { Link, useLocation, useSearchParams } from "react-router";
import { Fragment, useRef, useState } from "react";

import { usePopper } from "react-popper";
import { LogOut } from "lucide-react";
import { Popover, Transition } from "@headlessui/react";
// plane imports
import { API_BASE_URL } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { AuthService } from "@plane/services";
import { Avatar } from "@plane/ui";
import { getFileURL } from "@plane/utils/file";
// helpers
import { queryParamGenerator } from "@/helpers/query-param-generator";
// hooks

const authService = new AuthService();

export const UserAvatar = function UserAvatar() {
  const { pathname: pathName } = useLocation();
  const [searchParams] = useSearchParams();
  // query params
  const board = searchParams.get("board") || undefined;
  const labels = searchParams.get("labels") || undefined;
  const state = searchParams.get("state") || undefined;
  const priority = searchParams.get("priority") || undefined;
  const peekId = searchParams.get("peekId") || undefined;
  // hooks
  const { data: currentUser } = useQuery(currentUserOptions);
  const client = useQueryClient();
  const { setTheme } = useTheme();
  const formElement = useRef<HTMLFormElement>(null);
  const form = useForm({
    defaultValues: {},
    validators: { onSubmit: (): string | undefined => undefined },
    onSubmit: async () => {
      await authService.submitForm(formElement.current);
      client.clear();
      setTheme("system");
    },
  });
  const busy = useStore(form.store, (formState) => formState.isSubmitting);
  const error = useStore(form.store, (formState) => formState.errorMap.onSubmit);
  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: "bottom-end",
    modifiers: [
      {
        name: "offset",
        options: {
          offset: [0, 40],
        },
      },
    ],
  });

  // derived values
  const { queryParam } = queryParamGenerator({ peekId, board, state, priority, labels });

  return (
    <div className="relative mr-2">
      {currentUser?.id ? (
        <div>
          <Popover as="div">
            <Popover.Button as={Fragment}>
              <button ref={setReferenceElement} className="flex items-center gap-2 rounded-sm border border-subtle p-2">
                <Avatar
                  name={currentUser.display_name}
                  src={getFileURL(currentUser.avatar_url)}
                  shape="square"
                  size="sm"
                  showTooltip={false}
                />
                <h6 className="text-11 font-medium text-secondary">
                  {currentUser.display_name ||
                    [currentUser.first_name, currentUser.last_name].filter(Boolean).join(" ") ||
                    currentUser.email ||
                    "User"}
                </h6>
              </button>
            </Popover.Button>
            <Transition
              as={Fragment}
              enter="transition ease-out duration-200"
              enterFrom="opacity-0 translate-y-1"
              enterTo="opacity-100 translate-y-0"
              leave="transition ease-in duration-150"
              leaveFrom="opacity-100 translate-y-0"
              leaveTo="opacity-0 translate-y-1"
            >
              <Popover.Panel>
                <div
                  className="z-10 overflow-hidden rounded-sm border border-subtle bg-surface-1 p-1 shadow-raised-200"
                  ref={setPopperElement}
                  style={styles.popper}
                  {...attributes.popper}
                >
                  <form
                    ref={formElement}
                    method="POST"
                    action={`${API_BASE_URL}/auth/spaces/sign-out/`}
                    onSubmit={(event) => {
                      event.preventDefault();
                      form.setErrorMap({ onSubmit: undefined });
                      void form.handleSubmit().catch(() =>
                        form.setErrorMap({
                          onSubmit: { form: "Could not sign out. Please try again.", fields: {} },
                        })
                      );
                    }}
                  >
                    <input type="hidden" name="csrfmiddlewaretoken" value="" />
                    <input type="hidden" name="next_path" value={`${pathName}?${queryParam}`} />
                    <button
                      type="submit"
                      disabled={busy}
                      className="flex min-w-36 cursor-pointer items-center gap-2 rounded-sm p-2 text-13 whitespace-nowrap hover:bg-layer-transparent-hover"
                    >
                      <LogOut size={12} className="shrink-0 text-danger-primary" />
                      <div>Sign out</div>
                    </button>
                    {error && (
                      <p role="alert" className="p-2 text-13 text-danger-primary">
                        {error}
                      </p>
                    )}
                  </form>
                </div>
              </Popover.Panel>
            </Transition>
          </Popover>
        </div>
      ) : (
        <div className="flex-shrink-0">
          <Link to={`/?next_path=${pathName}?${queryParam}`}>
            <Button variant="secondary">Sign in</Button>
          </Link>
        </div>
      )}
    </div>
  );
};

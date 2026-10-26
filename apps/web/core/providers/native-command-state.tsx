/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useCallback, useMemo, useState } from "react";
import type { TCreateModalStoreTypes, TCreatePageModal } from "@plane/constants";
import { DEFAULT_CREATE_PAGE_MODAL_DATA, EPageAccess } from "@plane/constants";
import type { TProfileSettingsTabs } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import { PowerKCommandRegistry } from "../components/power-k/core/registry";
import type { IPowerKCommandRegistry } from "../components/power-k/core/registry";
import type { TPowerKCommandConfig, TPowerKContextType, TPowerKPageType } from "@/components/power-k/core/types";

type TCommandPaletteState = {
  isCreateProjectModalOpen: boolean;
  isCreateCycleModalOpen: boolean;
  isCreateModuleModalOpen: boolean;
  isCreateViewModalOpen: boolean;
  createPageModal: TCreatePageModal;
  isCreateIssueModalOpen: boolean;
  isDeleteIssueModalOpen: boolean;
  isBulkDeleteIssueModalOpen: boolean;
  createIssueStoreType: TCreateModalStoreTypes;
  createWorkItemAllowedProjectIds: string[] | undefined;
  profileSettingsModal: { activeTab: TProfileSettingsTabs | null; isOpen: boolean };
  getIsProjectListOpen: (projectId: string) => boolean;
  toggleCreateProjectModal: (value?: boolean) => void;
  toggleCreateCycleModal: (value?: boolean) => void;
  toggleCreateViewModal: (value?: boolean) => void;
  toggleCreatePageModal: (value?: TCreatePageModal) => void;
  toggleCreateIssueModal: (value?: boolean, storeType?: TCreateModalStoreTypes, allowedProjectIds?: string[]) => void;
  toggleCreateModuleModal: (value?: boolean) => void;
  toggleDeleteIssueModal: (value?: boolean) => void;
  toggleBulkDeleteIssueModal: (value?: boolean) => void;
  toggleProjectListOpen: (projectId: string, value?: boolean) => void;
  toggleProfileSettingsModal: (value: { activeTab?: TProfileSettingsTabs | null; isOpen?: boolean }) => void;
};

type TPowerKState = {
  isPowerKModalOpen: boolean;
  isShortcutsListModalOpen: boolean;
  activeContext: TPowerKContextType | null;
  activePage: TPowerKPageType | null;
  topNavInputRef: React.RefObject<HTMLInputElement | null> | null;
  topNavSearchInputRef: React.RefObject<HTMLInputElement | null> | null;
  setActiveContext: (context: TPowerKContextType | null) => void;
  setActivePage: (page: TPowerKPageType | null) => void;
  setTopNavInputRef: (ref: React.RefObject<HTMLInputElement | null> | null) => void;
  setTopNavSearchInputRef: (ref: React.RefObject<HTMLInputElement | null> | null) => void;
  togglePowerKModal: (value?: boolean) => void;
  toggleShortcutsListModal: (value?: boolean) => void;
};

type TPowerKRegistryState = {
  commandRegistry: IPowerKCommandRegistry;
  registerCommands: (commands: TPowerKCommandConfig[]) => () => void;
};

export const CommandPaletteContext = createContext<TCommandPaletteState | undefined>(undefined);
export const PowerKContext = createContext<TPowerKState | undefined>(undefined);
export const PowerKRegistryContext = createContext<TPowerKRegistryState | undefined>(undefined);

export function NativeCommandStateProvider({ children }: { children: React.ReactNode }) {
  const [isCreateProjectModalOpen, setCreateProjectModalOpen] = useState(false);
  const [isCreateCycleModalOpen, setCreateCycleModalOpen] = useState(false);
  const [isCreateModuleModalOpen, setCreateModuleModalOpen] = useState(false);
  const [isCreateViewModalOpen, setCreateViewModalOpen] = useState(false);
  const [createPageModal, setCreatePageModal] = useState<TCreatePageModal>(DEFAULT_CREATE_PAGE_MODAL_DATA);
  const [createIssue, setCreateIssue] = useState<{
    isOpen: boolean;
    storeType: TCreateModalStoreTypes;
    allowedProjectIds: string[] | undefined;
  }>({
    isOpen: false,
    storeType: EIssuesStoreType.PROJECT,
    allowedProjectIds: undefined,
  });
  const [isDeleteIssueModalOpen, setDeleteIssueModalOpen] = useState(false);
  const [isBulkDeleteIssueModalOpen, setBulkDeleteIssueModalOpen] = useState(false);
  const [profileSettingsModal, setProfileSettingsModal] = useState<{
    activeTab: TProfileSettingsTabs | null;
    isOpen: boolean;
  }>({ activeTab: "general", isOpen: false });
  const [projectListOpenMap, setProjectListOpenMap] = useState<Partial<Record<string, boolean>>>({});

  const toggleCreateProjectModal = useCallback((value?: boolean) => {
    setCreateProjectModalOpen((open) => value ?? !open);
  }, []);
  const toggleCreateCycleModal = useCallback((value?: boolean) => {
    setCreateCycleModalOpen((open) => value ?? !open);
  }, []);
  const toggleCreateModuleModal = useCallback((value?: boolean) => {
    setCreateModuleModalOpen((open) => value ?? !open);
  }, []);
  const toggleCreateViewModal = useCallback((value?: boolean) => {
    setCreateViewModalOpen((open) => value ?? !open);
  }, []);
  const toggleCreatePageModal = useCallback((value?: TCreatePageModal) => {
    setCreatePageModal((current) =>
      value
        ? { isOpen: value.isOpen, pageAccess: value.pageAccess || EPageAccess.PUBLIC }
        : { isOpen: !current.isOpen, pageAccess: EPageAccess.PUBLIC }
    );
  }, []);
  const toggleCreateIssueModal = useCallback(
    (value?: boolean, storeType?: TCreateModalStoreTypes, allowedProjectIds?: string[]) => {
      setCreateIssue((current) =>
        value !== undefined
          ? {
              isOpen: value,
              storeType: storeType || EIssuesStoreType.PROJECT,
              allowedProjectIds: allowedProjectIds ?? undefined,
            }
          : { isOpen: !current.isOpen, storeType: EIssuesStoreType.PROJECT, allowedProjectIds: undefined }
      );
    },
    []
  );
  const toggleDeleteIssueModal = useCallback((value?: boolean) => {
    setDeleteIssueModalOpen((open) => value ?? !open);
  }, []);
  const toggleBulkDeleteIssueModal = useCallback((value?: boolean) => {
    setBulkDeleteIssueModalOpen((open) => value ?? !open);
  }, []);
  const toggleProjectListOpen = useCallback((projectId: string, value?: boolean) => {
    setProjectListOpenMap((current) => ({
      ...current,
      [projectId]: value ?? !current[projectId],
    }));
  }, []);
  const getIsProjectListOpen = useCallback(
    (projectId: string) => Boolean(projectListOpenMap[projectId]),
    [projectListOpenMap]
  );
  const toggleProfileSettingsModal = useCallback(
    (value: { activeTab?: TProfileSettingsTabs | null; isOpen?: boolean }) => {
      setProfileSettingsModal((current) => ({ ...current, ...value }));
    },
    []
  );

  const commandPaletteValue = useMemo<TCommandPaletteState>(
    () => ({
      isCreateProjectModalOpen,
      isCreateCycleModalOpen,
      isCreateModuleModalOpen,
      isCreateViewModalOpen,
      createPageModal,
      isCreateIssueModalOpen: createIssue.isOpen,
      isDeleteIssueModalOpen,
      isBulkDeleteIssueModalOpen,
      createIssueStoreType: createIssue.storeType,
      createWorkItemAllowedProjectIds: createIssue.allowedProjectIds,
      profileSettingsModal,
      getIsProjectListOpen,
      toggleCreateProjectModal,
      toggleCreateCycleModal,
      toggleCreateViewModal,
      toggleCreatePageModal,
      toggleCreateIssueModal,
      toggleCreateModuleModal,
      toggleDeleteIssueModal,
      toggleBulkDeleteIssueModal,
      toggleProjectListOpen,
      toggleProfileSettingsModal,
    }),
    [
      createIssue,
      createPageModal,
      getIsProjectListOpen,
      isBulkDeleteIssueModalOpen,
      isCreateCycleModalOpen,
      isCreateModuleModalOpen,
      isCreateProjectModalOpen,
      isCreateViewModalOpen,
      isDeleteIssueModalOpen,
      profileSettingsModal,
      toggleBulkDeleteIssueModal,
      toggleCreateCycleModal,
      toggleCreateIssueModal,
      toggleCreateModuleModal,
      toggleCreatePageModal,
      toggleCreateProjectModal,
      toggleCreateViewModal,
      toggleDeleteIssueModal,
      toggleProfileSettingsModal,
      toggleProjectListOpen,
    ]
  );

  const [isPowerKModalOpen, setPowerKModalOpen] = useState(false);
  const [isShortcutsListModalOpen, setShortcutsListModalOpen] = useState(false);
  const [activeContext, setActiveContext] = useState<TPowerKContextType | null>(null);
  const [activePage, setActivePage] = useState<TPowerKPageType | null>(null);
  const [topNavInputRef, setTopNavInputRef] = useState<React.RefObject<HTMLInputElement | null> | null>(null);
  const [topNavSearchInputRef, setTopNavSearchInputRef] = useState<React.RefObject<HTMLInputElement | null> | null>(
    null
  );
  const [registeredCommands, setRegisteredCommands] = useState<
    { id: symbol; commands: TPowerKCommandConfig[] } | undefined
  >();
  const registerCommands = useCallback((commands: TPowerKCommandConfig[]) => {
    const registration = { id: Symbol("power-k-commands"), commands };
    setRegisteredCommands(registration);
    return () => {
      setRegisteredCommands((current) => (current?.id === registration.id ? undefined : current));
    };
  }, []);
  const togglePowerKModal = useCallback((value?: boolean) => {
    setPowerKModalOpen((open) => value ?? !open);
  }, []);
  const toggleShortcutsListModal = useCallback((value?: boolean) => {
    setShortcutsListModalOpen((open) => value ?? !open);
  }, []);
  const commandRegistry = useMemo(
    () => new PowerKCommandRegistry(registeredCommands?.commands ?? []),
    [registeredCommands]
  );
  const powerKValue = useMemo<TPowerKState>(
    () => ({
      isPowerKModalOpen,
      isShortcutsListModalOpen,
      activeContext,
      activePage,
      topNavInputRef,
      topNavSearchInputRef,
      setActiveContext,
      setActivePage,
      setTopNavInputRef,
      setTopNavSearchInputRef,
      togglePowerKModal,
      toggleShortcutsListModal,
    }),
    [
      activeContext,
      activePage,
      isPowerKModalOpen,
      isShortcutsListModalOpen,
      togglePowerKModal,
      toggleShortcutsListModal,
      topNavInputRef,
      topNavSearchInputRef,
    ]
  );
  const registryValue = useMemo<TPowerKRegistryState>(
    () => ({ commandRegistry, registerCommands }),
    [commandRegistry, registerCommands]
  );

  return (
    <CommandPaletteContext.Provider value={commandPaletteValue}>
      <PowerKContext.Provider value={powerKValue}>
        <PowerKRegistryContext.Provider value={registryValue}>{children}</PowerKRegistryContext.Provider>
      </PowerKContext.Provider>
    </CommandPaletteContext.Provider>
  );
}

import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import PATHS from "router/paths";

import { IConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
import { ApplePlatform, isAndroid } from "interfaces/platform";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";

import EndUserOSRequirementPreview from "./components/EndUserOSRequirementPreview";
import TurnOnMdmMessage from "../../../components/TurnOnMdmMessage/TurnOnMdmMessage";
import CurrentVersionSection from "./components/CurrentVersionSection";
import TargetSection from "./components/TargetSection";
import { parseOSUpdatesCurrentVersionsQueryParams } from "./components/CurrentVersionSection/CurrentVersionSection";

export type OSUpdatesSupportedPlatform = ApplePlatform | "windows";

export type OSUpdatesTargetPlatform = OSUpdatesSupportedPlatform | "android"; // used for displaying "coming soon" messaging

const baseClass = "os-updates";

const getDefaultSelectedPlatform = (
  appConfig: IConfig | null
): OSUpdatesTargetPlatform => {
  // We dont have the data ready yet so we default to mac.
  // This is usually when the users first comes to this page.
  if (appConfig === null) return "darwin";

  // if the mac mdm is enable and configured we check the app config to see if
  // the mdm for mac is enabled. If it is, it does not matter if windows is
  // enabled and configured and we will always return "mac".
  return appConfig.mdm.enabled_and_configured ? "darwin" : "windows"; // TODO(android): adjust this when android is supported
};

interface IOSUpdates {
  router: InjectedRouter;
  teamIdForApi: number;
  queryParams: ReturnType<typeof parseOSUpdatesCurrentVersionsQueryParams>;
}

const OSUpdates = ({ router, teamIdForApi, queryParams }: IOSUpdates) => {
  const {
    isPremiumTier,
    isGlobalAdmin,
    isTeamAdmin,
    config,
    setConfig,
  } = useContext(AppContext);

  const [
    selectedPlatformTab,
    setSelectedPlatformTab,
  ] = useState<OSUpdatesTargetPlatform | null>(null);

  const {
    isFetching: isFetchingConfig,
    isLoading: isLoadingConfig,
    refetch: refetchAppConfig,
  } = useQuery<IConfig, Error>(["config"], () => configAPI.loadAll(), {
    refetchOnWindowFocus: false,
    onSuccess: (data) => setConfig(data), // update the app context with the refetched config
    enabled: false, // this is disabled as the config is already fetched in App.tsx
  });

  const {
    data: teamConfig,
    isFetching: isFetchingTeamConfig,
    isLoading: isLoadingTeam,
    refetch: refetchTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team-config", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      refetchOnWindowFocus: false,
      enabled: !!teamIdForApi && (isGlobalAdmin || isTeamAdmin),
      select: (data) => data.team,
    }
  );

  if (!isGlobalAdmin && !isTeamAdmin) {
    router.replace(PATHS.CONTROLS_OS_SETTINGS);
  }

  // Not premium shows premium message
  if (!isPremiumTier) {
    return (
      <PremiumFeatureMessage
        className={`${baseClass}__premium-feature-message`}
      />
    );
  }

  if (isLoadingConfig || isLoadingTeam) return <Spinner />;

  // FIXME: Handle error states for app config and team config (need specifications for this).

  // mdm is not enabled for mac or windows.

  if (
    !config?.mdm.enabled_and_configured &&
    !config?.mdm.windows_enabled_and_configured
  ) {
    return <TurnOnMdmMessage router={router} />;
  }

  // If the user has not selected a platform yet, we default to the platform that
  // is enabled and configured.
  const selectedPlatform =
    selectedPlatformTab || getDefaultSelectedPlatform(config);

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely encourage the installation of software updates on hosts
        assigned to this team.
      </p>
      <div className={`${baseClass}__content`}>
        <div className={`${baseClass}__current-version-container`}>
          <CurrentVersionSection
            router={router}
            currentTeamId={teamIdForApi}
            queryParams={queryParams}
          />
        </div>
        <div className={`${baseClass}__target-container`}>
          <TargetSection
            key={teamIdForApi} // if the team changes, remount the target section
            appConfig={config}
            currentTeamId={teamIdForApi}
            isFetching={isFetchingConfig || isFetchingTeamConfig}
            selectedPlatform={selectedPlatform}
            teamConfig={teamConfig}
            onSelectPlatform={setSelectedPlatformTab}
            refetchAppConfig={refetchAppConfig}
            refetchTeamConfig={refetchTeamConfig}
          />
        </div>
        {!isAndroid(selectedPlatform) && (
          <div className={`${baseClass}__nudge-preview`}>
            <EndUserOSRequirementPreview platform={selectedPlatform} />
          </div>
        )}
      </div>
    </div>
  );
};

export default OSUpdates;

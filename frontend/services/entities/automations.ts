/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IResetAutomationIds {
  team_ids?: number[];
  policy_ids?: number[];
}

export default {
  resetAutomations: (resetAutomationIds: IResetAutomationIds) => {
    const { RESET_AUTOMATIONS } = endpoints;

    return sendRequest("POST", RESET_AUTOMATIONS, resetAutomationIds);
  },
};

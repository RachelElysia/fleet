---
name: 🎟  Story
about: Specify an iterative change to the Fleet product.  (e.g. "As a user, I want to sign in with SSO.")
title: ''
labels: 'story'
assignees: ''

---

<!-- **This issue's remaining effort can be completed in ≤1 sprint.  It will be valuable even if nothing else ships.**
It is [planned and ready](https://fleetdm.com/handbook/company/development-groups#making-changes) to implement.  It is on the proper kanban board. -->


## Goal

| User story  |
|:---------------------------------------------------------------------------|
| As a _________________________________________,
| I want to _________________________________________
| so that I can _________________________________________.

## Roadmap item

<!-- Paste the roadmap item, from the current quarter's roadmap preview article, this story contributes to. Put "None" if it doesn't contribute to a roadmap item. -->

## Original requests

<!-- Insert the link to the feature request that this story contributes to and add the feature request as this story's parent issue. Put "None" if it doesn't contribute to a request. For customer requests, add the `customer-xyz` label(s). -->

## Context
- Product Designer: _________________________ <!-- Who is the Product Designer to contact if folks have questions about the UI, CLI, or API changes? -->
- Engineer: _________________________ <!-- Who is the Engineer to contact if folks have technical questions? -->
  
<!--
What else should contributors [keep in mind](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) when working on this change?  (Optional.)
1. 
2. 
-->

## Changes

### Product
- [ ] UI changes: TODO <!-- Insert the link to the relevant Figma cover page. If there are substantial UI changes at one of Fleet's breakpoints (480, 768, 1024, 1280, and 1440px), make sure wireframes show the UI at the relevant breakpoint(s). Put "No changes" if there are no changes to the user interface. -->
- [ ] CLI (fleetctl) usage changes: TODO <!-- Insert the link to the relevant Figma cover page. Put "No changes" if there are no changes to the CLI. -->
- [ ] YAML changes: TODO <!-- Specify changes in the YAML files doc page as a PR to the reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting Put "No changes" if there are no changes necessary. -->
- [ ] REST API changes: TODO <!-- Specify changes in the the REST API doc page as a PR to reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting Put "No changes" if there are no changes necessary. Move this item to the engineering list below if engineering will design the API changes. -->
- [ ] Fleet's agent (fleetd) changes: TODO <!-- Specify changes to fleetd. If the change requires a new Fleet (server) version, consider specifying to only enable this change in new Fleet versions. Put "No changes" if there are no changes necessary. -->
- [ ] GitOps mode UI changes: TODO <!-- Specify UI changes for read-only GitOps mode. Put "No changes" if there are no changes necessary. -->
- [ ] GitOps generation changes: TODO <!-- Specify changes to results from the fleetctl generate-gitops command. Put "No changes" if there are no changes necessary. -->
- [ ] Activity changes: TODO <!-- Specify changes to the Audit log page in the contributor docs. Put "No changes" if there are no changes necessary. -->
- [ ] Permissions changes: TODO <!-- Specify changes in the Manage access doc page as a PR to the reference docs release branch. If doc changes aren't necessary, explicitly mention no changes to the doc page. Put "No changes" if there are no permissions changes. -->
- [ ] Changes to paid features or tiers: TODO  <!-- Specify changes in pricing-features-table.yml as a PR to reference docs release branch. Specify "Fleet Free" and/or "Fleet Premium" if there are no changes to the pricing page necessary. -->
- [ ] My device and fleetdm.com/better changes: TODO <!-- If there are changes to the personal information Fleet can see on end user workstations, make sure wireframes include changes to the My device page. Also, specify changes as a PR to the fleetdm.com/better (aka Transparency page). Put "No changes" if there are no changes necessary. -->
- [ ] Other reference documentation changes: TODO <!-- Any other reference doc changes? Specify changes as a PR to reference docs release branch. Put "No changes" if there are no changes necessary. -->
- [ ] First draft of test plan added
- [ ] Once shipped, requester has been notified
- [ ] Once shipped, dogfooding issue has been filed

### Engineering
- [ ] Test plan is finalized
- [ ] Contributor API changes: TODO <!-- Specify changes in the the Contributor API doc page as a PR to reference docs release branch following the guidelines in the handbook here: https://fleetdm.com/handbook/product-design#drafting Put "No changes" if there are no changes necessary. -->
- [ ] Feature guide changes: TODO <!-- Specify if a new feature guide is required at fleetdm.com/guides, or if a previous guide should be updated to reflect feature changes. -->
- [ ] Database schema migrations: TODO <!-- Specify what changes to the database schema are required. (This will be used to change migration scripts accordingly.) Remove this checkbox if there are no changes necessary. -->
- [ ] Load testing: TODO  <!-- List any required scalability testing to be conducted.  Remove this checkbox if there is no scalability testing required. -->
- [ ] Load testing/osquery-perf improvements: TODO <-- List, or link a subtask for, any osquery-perf or load test environment changes required to comprehensively load test this story if load testing is needed. -->

> ℹ️  Please read this issue carefully and understand it.  Pay [special attention](https://fleetdm.com/handbook/company/development-groups#developing-from-wireframes) to UI wireframes, especially "dev notes".

## QA

### Risk assessment

- Requires load testing: TODO <!-- User story has performance implications that require load testing. Otherwise, remove this item. -->
- Risk level: Low / High TODO <!-- Choose one. Consider: Does this change come with performance risks?  Any risk of accidental log spew? Any particular regressions to watch out for?  Any potential compatibility issues, even if it's not technically a breaking change? -->
- Risk description: TODO <!-- If the risk level is high, explain why. If low, remove. -->

### Test plan
<!-- Add detailed manual testing steps for all affected user roles. -->

> Make sure to go through [the list](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/guides/ui/design-qa-considerations.md) and consider all events that might be related to this story, so we catch edge cases earlier.

1. TODO
2. TODO
3. TODO

<!-- Consider: Do the steps above apply to all global access roles, including admin, maintainer, observer, observer+, and GitOps?  Do the steps above apply to all team-level access roles?  If not, write the steps used to test each variation.
-->

### Testing notes
<!-- Any additional testing notes relevant to this story or tools required for testing. -->

### Confirmation
<!-- The engineer responsible for implementing this user story completes the test plan before moving to the "Ready for QA" column. -->

1. [ ] Engineer: Added comment to user story confirming successful completion of test plan.
2. [ ] QA: Added comment to user story confirming successful completion of test plan.

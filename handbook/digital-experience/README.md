# Digital Experience 

This page details processes specific to working [with](#contact-us) and [within](#responsibilities) this department.


## Team

| Role                            | Contributor(s)
|:--------------------------------|:----------------------------------------------------------------------|
| [CEO](https://fleetdm.com/handbook/company/leadership#ceo-flaws) | [Mike McNeil](https://www.linkedin.com/in/mikermcneil) _([@mikermcneil](https://github.com/mikermcneil))_
| Head of People / HR / Legal     | <sup><sub> See [CEO](https://www.fleetdm.com/handbook/digital-experience#team) <sup><sub>
| Head of Digital Experience      | [Sam Pfluger](https://www.linkedin.com/in/sampfluger88/) _([@sampfluger88](https://github.com/sampfluger88))_ 
| GTM Engineer | [Isabell Reedy](https://www.linkedin.com/in/isabell-reedy-202aa3123/) _([@ireedy](https://github.com/ireedy))_
| Head of Design                  | [Mike Thomas](https://www.linkedin.com/in/mike-thomas-52277938) _([@mike-j-thomas](https://github.com/mike-j-thomas))_
| Software Engineer               | [Eric Shaw](https://www.linkedin.com/in/eric-shaw-1423831a9/) _([@eashaw](https://github.com/eashaw))_
| Apprentice                      | [Irena Reedy](https://www.linkedin.com/in/irena-reedy-520ab9354/) _([@irenareedy](https://github.com/irenareedy))_, [Onasis Munro](https://www.linkedin.com/in/onasismunro/) _([@onasismunro](https://github.com/onasismunro))_


## Contact us

- To **make a request** of this department, [create an issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-digital-experience&projects=&template=1-custom-request.md&title=) and a team member will get back to you within one business day (If urgent, mention a [team member](#team) in the [#g-digital-experience](https://fleetdm.slack.com/archives/C058S8PFSK0) Slack channel.
  - Any Fleet team member can [view the kanban board](https://github.com/orgs/fleetdm/projects/65) for this department, including pending tasks and the status of new requests.
  - Please **use issue comments and GitHub mentions** to communicate follow-ups or answer questions related to your request.


## Responsibilities

The Digital Experience department is directly responsible for the culture, training, framework, content design, and technology behind Fleet's remote work culture, including fleetdm.com, the handbook, issue templates, UI style guides, internal tooling, Zapier flows, Docusign templates, key spreadsheets, contracts, compliance, receiving and responding to legal notices, SOC2, deal desk, project management processes, human resources, benefits, opening positions, compensation planning, onboarding, and offboarding.

> Commission planning, taxes, state unemployment insurance filings, business insurance, Delaware registered agent and franchise taxes, virtual mailbox, company phone number, and other adjacent areas of responsibility are run by [the Finance department](https://fleetdm.com/handbook/finance)._

> If a user story involves only changes to fleetdm.com, without changing the core product, then that user story is prioritized, drafted, implemented, and shipped by the [Digital Experience](https://fleetdm.com/handbook/digital-experience) department.  Otherwise, if the story **also** involves changes to the core product **as well as** fleetdm.com, then that user story is prioritized, drafted, implemented, and shipped by [the other relevant product group](https://fleetdm.com/handbook/company/product-groups#current-product-groups), and not by `#g-digital-experience`._


### Respond to a "Contact us" submission

1. Check the [_from-prospective-customers](https://fleetdm.slack.com/archives/C01HE9GQW6B) Slack channel for "Contact us" submissions. 
2. Mark submission as seen with the "👀" emoji.
3. Within 4 business hours, use the [_from-prospective-customers workflow (private Google doc)](https://docs.google.com/spreadsheets/d/1-wsYunAfr-BQZMBYizY4TMavi3X071D5KZ3mCYX4Uqs/edit?gid=0#gid=0) to respond to general asks. Follow the "High-level workflow" to understand how to respond and who to loop into the conversation. 
4. Answer any technical questions to the best of your ability. If you are unable to answer a technical/product question, ask a Solutions Consultant in `#help-solutions-consulting`. If an SC is unavailable, post in `#g-mdm`or `#g-endpoint-ops`and notify @on-call.
5. Mark the Slack message as complete with the "✅" emoji.

> For any support-related questions, forward the submission to [Fleet's support team](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit#heading=h.wqalwz1je6rq).


### Prepare "Let's get you set up!" meeting notes

Before each group call, copy the attendees from the "Lets get you set up! (group office hours)" calendar event and paste them into the correct section of the ["Let's get you set up!" meeting notes](https://docs.google.com/document/d/1rlvueDlTqiz0cyH426nVL6LXpv9MWDUtXW6YiMT3oK8/edit?tab=t.0#heading=h.l967l3n9mjnd), be sure to use the format listed in the doc. 


### Manage duplicate accounts in CRM

1. Navigate to ["Ω Possible duplicate accounts" report](https://fleetdm.lightning.force.com/lightning/r/Report/00OUG000001FA1h2AG/view?queryScope=userFolders).
2. Verify that each potential duplicate account is indeed a duplicate of the account's it has been paired with.
3. Open duplicate accounts and compare duplicate accounts to select the best account to "Use as principal" (the account all other duplicates will be merged into). Consider the following:
  - Is there an open opportunity on any of the accounts? If so, this is your "principal" account.
  - Do any of the accounts not have contacts? If no contacts found on the account and no significant activity, delete the account. 
  - Do any of these accounts have activity that the others don't have (e.g. a rep sent an email or logged a call)? Be sure to preserve the maximum amount of historical activity on the principal account.
4. Click view duplicates, select all relevant accounts that appear. Click next.
5. Select the best and most up-to-date data to combine into the single principal account.

> Do *NOT* change account owners if you can help it during this process. For "non-sales-ready" accounts default to the Integrations Admin. If the account is owned by an active user, be sure they maintain ownership of the principal account. 

6. YOU CAN NOT UNDO THIS NEXT PART! Click next, click merge. 
7. Verify that the principal account details match exactly what is on LinkedIn. The end result should be as follows:
  - LinkedIn company url
  - Website
  - Employees


### QA a change to fleetdm.com

Each PR to the website is manually checked for quality and tested before going live on fleetdm.com. To test any change to fleetdm.com

1. Write clear step-by-step instructions to confirm that the change to the fleetdm.com functions as expected and doesn't break any possible automation. These steps should be simple and clear enough for anybody to follow.

2. [View the website locally](https://fleetdm.com/handbook/digital-experience#test-fleetdm-com-locally) and follow the QA steps in the request ticket to test changes.

3. Check the change in relation to all breakpoints and [browser compatibility](https://fleetdm.com/handbook/digital-experience#check-browser-compatibility-for-fleetdm-com), Tests are carried out on [supported browsers](https://fleetdm.com/docs/using-fleet/supported-browsers) before website changes go live.


### Test fleetdm.com locally 

When making changes to the Fleet website, you can test your changes by running the website locally. To do this, you'll need the following:

- A local copy of the [Fleet repo](https://github.com/fleetdm/fleet).
- [Node.js](https://nodejs.org/en/download/)
- (Optional) [Sails.js](https://sailsjs.com/) installed globally on your machine (`npm install sails -g`)

Once you have the above follow these steps:

1. Open your terminal program, and navigate to the `website/` folder of your local copy of the Fleet repo.
    
    > Note: If this is your first time running this script, you will need to run `npm install` inside of the website/ folder to install the website's dependencies.


2. Run the `build-static-content` script to generate HTML pages from our Markdown and YAML content.
  - **With Node**, you will need to use `node ./node_modules/sails/bin/sails run build-static-content` to execute the script.
  - **With Sails.js installed globally** you can use `sails run build-static-content` to execute the script.

    > When this script runs, the website's configuration file ([`website/.sailsrc`](https://github.com/fleetdm/fleet/blob/main/website/.sailsrc)) will automatically be updated with information the website uses to display content built from Markdown and YAML. Changes to this file should never be committed to the GitHub repo. If you want to exclude changes to this file in any PRs you make, you can run this terminal command in your local copy of the Fleet repo: `git update-index --assume-unchanged ./website/.sailsrc`.
    
    > Note: You can run `npm run start-dev` in the `website/` folder to run the `build-static-content` script and start the website server with a single command.

3. Once the script is complete, start the website server:
  - **With Node.js:** start the server by running `node ./node_modules/sails/bin/sails lift`
  - **With Sails.js installed globally:** start the server by running `sails lift`.

4. When the server has started, the Fleet website will be available at [http://localhost:2024](http://localhost:2024)
    
  > **Note:** Some features, such as self-service license dispenser and account creation, are not available when running the website locally. If you need help testing features on a local copy, reach out to `@eashaw` in the [#g-digital-experience](https://fleetdm.slack.com/archives/C058S8PFSK0) channel on Slack.


### Check production dependencies of fleetdm.com

Every week, we run `npm audit --only=prod` to check for vulnerabilities on the production dependencies of fleetdm.com. Once we have a solution to configure GitHub's Dependabot to ignore devDependencies, this [manual process](https://www.loom.com/share/153613cc1c5347478d3a9545e438cc97?sid=5102dafc-7e27-43cb-8c62-70c8789e5559) can be replaced with Dependabot.


### Respond to a 5xx error on fleetdm.com

Production systems can fail for various reasons, and it can be frustrating to users when they do, and customer experience is significant to Fleet. In the event of system failure, Fleet will:
- investigate the problem to determine the root cause.
- identify affected users.
- escalate if necessary.
- understand and remediate the problem.
- notify impacted users of any steps they need to take (if any).  If a customer paid with a credit card and had a bad experience, default to refunding their money.
- Conduct an incident post-mortem to determine any additional steps we need (including monitoring) to take to prevent this class of problems from happening in the future.


### Check browser compatibility for fleetdm.com

A [browser compatibility check](https://www.loom.com/share/4b1945ccffa14b7daca8ab9546b8fbb9?sid=eaa4d27a-236b-426d-a7cb-9c3bdb2c8cdc) of [fleetdm.com](https://fleetdm.com/) should be carried out monthly to verify that the website looks and functions as expected across all [supported browsers](https://fleetdm.com/docs/using-fleet/supported-browsers).

- We use [BrowserStack](https://www.browserstack.com/users/sign_in) (logins can be found in [1Password](https://start.1password.com/open/i?a=N3F7LHAKQ5G3JPFPX234EC4ZDQ&v=3ycqkai6naxhqsylmsos6vairu&i=nwnxrrbpcwkuzaazh3rywzoh6e&h=fleetdevicemanagement.1password.com)) for our cross-browser checks.
- Check for issues against the latest version of Google Chrome (macOS). We use this as our baseline for quality assurance.
- Document any issues in GitHub as a [bug](https://github.com/fleetdm/fleet/issues/new?assignees=&labels=bug%2C%3Areproduce&template=bug-report.md&title=), and assign them for fixing.
- If in doubt about anything regarding design or layout, please reach out to the [Head of Design](https://fleetdm.com/handbook/digital-experience#team).


<!-- Commenting this out as we don't have any planned landing pages in the future see: https://github.com/fleetdm/fleet/issues/21117
### Generate a new landing page

Experimental pages are short-lived, temporary landing pages intended for a small audience. All experiments and landing pages need to go through the standard [drafting process](https://fleetdm.com/handbook/company/product-groups#making-changes) before they are created.

Website experiments and landing pages live behind `/imagine` url. Which is hidden from the sitemap and intended to be linked to from ads and marketing campaigns. Design experiments (flyers, swag, etc.) should be limited to small audiences (less than 500 people) to avoid damaging the brand or confusing our customers. In general, experiments that are of a design nature should be targeted at prospects and random users, never targeted at our customers.

Some examples of experiments that would live behind the `/imagine` url:
- A flyer for a meetup "Free shirt to the person who can solve this riddle!"
- A landing page for a movie screening presented by Fleet
- A landing page for a private event
- A landing page for an ad campaign that is running for 4 weeks.
- An A/B test on product positioning
- A giveaway page for a conference
- Table-top signage for a conference booth or meetup

The Fleet website has a built-in landing page generator that can be used to quickly create a new page that lives under the /imagine/ url.

To generate a new page, you'll need: 

- A local copy of the [Fleet repo](https://github.com/fleetdm/fleet).
- [Node.js](https://nodejs.org/en/download/)
- (Optional) [Sails.js](https://sailsjs.com/) installed globally on your machine (`npm install sails -g`)

1. Open your terminal program, and navigate to the `website/` folder of your local copy of the Fleet repo.
    
    > Note: If this is your first time running the website locally, you will need to run `npm install` inside of the website/ folder to install the website's dependencies.

2. Call the `landing-page` generator by running `node ./node_modules/sails/bin/sails generate landing-page [page-name]`, replacing `[page-name]` with the kebab-cased name (words seperated by dashes `-`) of your page.

3. After the files have been generated, you'll need to manually update the website's routes. To do this, copy and paste the generated route for the new page to the "Imagine" section of `website/config/routes.js`.

4. Next you need to update the stylesheets so that the page can inherit the correct styles. To do this, copy and paste the generated import statement to the "Imagine" section of `website/assets/styles/importer.less`.

5. Start the website by running `node ./node_modules/sails/bin/sails lift` (or `sails lift` if you have Sails installed globally). The new landing page will be availible at `http://localhost:1337/imagine/{page-name}`.

6. Replace the lorum ipsum and placeholder images on the generated page with the page's real content, and add a meta description and title by changing the `pageTitleForMeta` and `pageDescriptionForMeta in the page's `locals` in `website/config/routes.js`.
-->

### Check for new versions of osquery schema

When a new version of osquery is released, the Fleet website needs to be updated to use the latest version of the osquery schema. To do this, we update the website's `versionOfOsquerySchemaToUseWhenGeneratingDocumentation` configuration variable in [website/config/custom.js](https://github.com/fleetdm/fleet/blob/6eb6884c4f02dc24b49f394abe9dde5fd1875c55/website/config/custom.js#L327). The osquery schema is combined with Fleet's [YAML overrides](https://github.com/fleetdm/fleet/tree/main/schema/tables) to generate the [JSON schema](https://github.com/fleetdm/fleet/blob/main/schema/osquery_fleet_schema.json) used by the query side panel in Fleet, as well as Fleetdm.com's [osquery table documentation](/tables).

> Note: The version number used in the `versionOfOsquerySchemaToUseWhenGeneratingDocumentation` variable must correspond to a version of the JSON osquery schema in the [osquery/osquery-site repo](https://github.com/osquery/osquery-site/tree/main/src/data/osquery_schema_versions).


### Restart Algolia manually

At least once every hour, an Algolia crawler reindexes the Fleet website's content. If an error occurs while the website is being indexed, Algolia will block our crawler and respond to requests with this message: `"This action cannot be executed on a blocked crawler"`.

When this happens, you'll need to manually start the crawler in the [Algolia crawler dashboard](https://crawler.algolia.com/admin/) to unblock it. 
You can do this by logging into the crawler dashboard using the login saved in 1password and clicking the "Restart crawling" button on our crawler's "overview" page](https://crawler.algolia.com/admin/crawlers/497dd4fd-f8dd-4ffb-85c9-2a56b7fafe98/overview).

No further action is needed if the crawler successfully reindexes the Fleet website. If another error occurs while the crawler is running, take a screenshot of the error and add it to the GitHub issue created for the alert and @mention `eashaw` for help.


### Re-run the "Deploy Fleet Website" action

If the action fails, please complete the following steps:
1. Head to the fleetdm-website app in the [Heroku dashboard](https://heroku.com) and select the "Activity" tab.
2. Select "Roll back to here" on the second to most recent deploy.
3. Head to the fleetdm/fleet GitHub repository and re-run the Deploy Fleet Website action. 


### Update a company brand front

Fleet has several brand fronts that need to be updated from time to time. Check each [brand front](https://docs.google.com/spreadsheets/d/1c15vwMZytpCLHUdGvXxi0d6WGgPcQU1UBMniC1F9oKk/edit?gid=0#gid=0) for consistency and update as needed with the following: 
- The current pitch, found in the blurbs section of the [🎐 Why Fleet?](https://docs.google.com/document/d/1E0VU4AcB6UTVRd4JKD45Saxh9Gz-mkO3LnGSTBDLEZo/edit#heading=h.uovxedjegxdc) doc. 
- The current [brand imagery](https://www.figma.com/design/1J2yxqH8Q7u8V7YTtA1iej/Social-media-(logos%2C-covers%2C-banners)?node-id=3962-65895). Check this [Loom video](https://www.loom.com/share/4432646cc9614046aaa4a74da1c0adb5?sid=2f84779f-f0bd-4055-be69-282c5a16f5c5) for more info.


<!--
### Update the host count of a premium subscription

When a self-service license dispenser customer reaches out to upgrade a license via the contact form, a member of the [Marketing department](https://fleetdm.com/handbook/marketing) will create a confidential issue detailing the request and add it to the new requests column of [Digital Experience kanban board](https://github.com/fleetdm/confidential/issues#workspaces/g-digital-experience-6451748b4eb15200131d4bab/board). A member of this team will then log into Stripe using the shared login, and upgrade the customer's subscription.

To update the host count on a user's subscription:

1. Log in to the [Stripe dashboard](https://dashboard.stripe.com/dashboard) and search for the customer's email address.
2. Click on their subscription and select the "Update subscription" option in the "Actions" dropdown
3. Update the quantity of the user's subscription to be their desired host count.
4. Turn the "Proration charges" option on and select the "Charge proration amount immediately" option.
5. Under "Payment" select "Email invoice to the customer", and set the payment due date to be 15 days, and make sure the "Invoice payment page" option is checked.
6. Select "Update subscription" to send the user an updated invoice for their subscription. Once the customer pays their new invoice, the Fleet website will update the user's subscription and generate a new Fleet Premium license with an updated host count.
7. Let the person who created the request know what actions were taken so they can communicate them to the customer.


### Change customer credit card number

You can help a Premium license dispenser customers change their credit card by directing them to their [account dashboard](https://fleetdm.com/customers/dashboard). On that page, the customer can update their billing card by clicking the pencil icon next to their billing information.


### Cancel a Fleet Premium subscription

Use the following steps to cancel a Fleet Premium subscription:
1. Log into [Stripe](https://dashboard.stripe.com/dashboard) (login in 1Password) and paste the customer's email they used to sign up in the search bar at the top of the page.
2. Select the subscription related to the email and use the "Actions" drop-down to "Cancel immediately".
3. Reach out to the community member (using the [correct email template](https://docs.google.com/document/d/1D02k0tc5v-sEJ4uahAouuqnvZ6phxA_gP-IqmkBdMTE/edit#heading=h.vw9mkh5e9msx)) and let them know their subscription was canceled.
-->


### Register a domain for Fleet

Domain name registrations are handled through Namecheap. Access is managed via 1Password.


### Purchase a SaaS tool

When procuring SaaS tools and services, analyze the purchase of these subscription services look for these way to help the company:
- Get product demos whenever possible.  Does the product do what it's supposed to do in the way that it is supposed to do it?
- Avoid extra features you don't need, and if they're there anyway, avoid using them.
- Data portability: is it possible for Fleet to export it's data if we stop using it? Is it easy to pull that data in an understandable format?
- Programability: Does it have a publicly documented legible REST API that requires at most a single API token?
- Intentionality: The product fits into other tools and processes that Fleet uses today. Avoid [unintended consequences](https://en.wikipedia.org/wiki/Midas). The tool will change to fit the company, or we won't use it. 


### Secure company-issued equipment for a team member

As soon as an offer is accepted, Fleet provides laptops and YubiKey security keys for core team members to use while working at Fleet. The IT engineer will work with the new team member to get their equipment requested and shipped to them on time.

- [**Check the "📦 Warehouse" team in dogfood**](https://dogfood.fleetdm.com/dashboard?team_id=279) before purchasing any equipment including laptops, to ensure we efficiently [utilize existing assets before spending money](https://fleetdm.com/handbook/company/why-this-way#why-spend-less). If Fleet IT warehouse inventory can meet the needs of the request, file a [warehouse request](https://github.com/fleetdm/confidential/issues/new?assignees=sampfluger88&labels=%23g-digital-experience&projects=&template=warehouse-request.md&title=%F0%9F%92%BB+Warehouse+request).

- Apple computers shipping to the United States and Canada are ordered using the Apple [eCommerce Portal](https://ecommerce2.apple.com/asb2bstorefront/asb2b/en/USD/?accountselected=true), or by contacting the business team at an Apple Store or contacting the online sales team at [800-854-3680](tel:18008543680). The IT engineer can arrange for same-day pickup at a store local to the Fleetie if needed.
  - **Note:** Most Fleeties use 16-inch MacBook Pros. Team members are free to choose any laptop or operating system that works for them, as long as the price [is within reason](https://www.fleetdm.com/handbook/communications#spending-company-money). 

  - When ordering through the Apple eCommerce Portal, look for a banner with *Apple Store for FLEET DEVICE MANAGEMENT | Welcome [Your Name].* Hovering over *Welcome* should display *Your Profile.* If Fleet's account number is displayed, purchases will be automatically made available in Apple Business Manager (ABM).

- Apple computers for Fleeties in other countries should be purchased through an authorized reseller to ensure the device is enrolled in ADE. In countries that Apple does not operate or that do not allow ADE, work with the authorized reseller to find the best solution, or consider shipping to a US based Fleetie and then shipping on to the teammate. 

 > A 3-year AppleCare+ Protection Plan (APP) should be considered default for Apple computers >$1500. Base MacBook Airs, Mac minis, etc. do not need APP unless configured beyond the $1500 price point. APP provides 24/7 support, and global repair coverage in case of accidental screen damage or liquid spill, and battery service.

 - Order a pack of two [YubiKey 5C NFC security keys](https://www.yubico.com/product/yubikey-5-series/yubikey-5c-nfc/) for new team member, shipped to them directly.

- Include delivery tracking information when closing the support request so the new employee can be notified.


### Process incoming equipment

Upon receiving any device, follow these steps to process incoming equipment.
1. Find the device in ["🍽️ Dogfood"](https://dogfood.fleetdm.com/dashboard) to confirm the correct equipment was received.
2. Visibly inspect equipment and all related components (e.g. laptop charger) for damage.
3. Remove any stickers and clean devices and components.
4. Using the device's charger, plug in the device.
5. Using your company laptop, navigate to the host in dogfood, and click `actions` » `Unlock` and copy the unlock code. 
6. Turn on the device and enter the unlock code.
7. If the previous user has not wiped the device, navigate to the host in dogfood, and click `actions` » `wipe` and wait until the device is finished and restarts.

**If you need to manually recover a device or reinstall macOS**
1. Enter recovery mode using the [appropriate method](https://support.apple.com/en-us/HT204904).
2. Connect the device to WIFI.
3. Using the "Recovery assistant" tab (In the top left corner), select "Delete this Mac".
4. Follow the prompts to activate the device and reinstall the appropriate version of macOS.
> If you are prevented from completing the steps above, create a ["💻 IT support issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-digital-experience&projects=&template=1-custom-request.md&title=) for IT, for the device to be scheduled for troubleshooting and remediation. Please note in the issue where you encountered blockers to completing the steps.


### Ship approved equipment

Once the Digital Experience department approves inventory to be shipped from Fleet IT, follow these step to ship the equipment.
1. Compare the equipment request issue with the ["📦 Warehouse" team](https://dogfood.fleetdm.com/settings/teams/users?team_id=279) and verify physical inventory.
2. Plug in the device and ensure inventory has been correctly processed and all components are present (e.g. charger cord, power converter).
3. Package equipment for shipment and include Yubikeys (if requested).
4. Change the "host" info to reflect the new user.
  - If you encounter any issues, repeat the [process incoming equipment steps](https://fleetdm.com/handbook/digital-experience#process-incoming-equipment). If problems persist, create a ["💻 IT support issue](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-digital-experience&projects=&template=1-custom-request.md&title=) for IT to troubleshoot the device.
6. Ship via FedEx to the address listed in the equipment request.
7. Add a comment to the equipment request issue, at-mentioning the requestor with the FedEx tracking info and close the issue.


### Update personnel details

When a Fleetie, consultant or advisor requests an update to their personnel details (name, location, phone, etc), follow these steps to ensure accurate representation across systems.
1. Team member submits a [💼 Teammate relocation](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-digital-experience&projects=&template=x-teammate-relocation.md) to update their personnel details (or Digital Experience team creates if the request comes via email or is sensitive and needs a classified issue).
    - If change is for a primary identification or contact method, ask for evidence of change and capture in [employee's personnel file](https://drive.google.com/drive/folders/1UL7o3BzkTKnpvIS4hm_RtbOilSABo3oG?usp=drive_link).
2. Digital Experience makes change to HRIS (Gusto or Plane) to reflect change. 
    - Note: if making the change requires follow-up steps, resolve those steps to action the change.
3. Once change is effected in HRIS, Digital Experience makes changes to ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) spreadsheet.
4. If required, Digital Experience makes any relevant changes to [Fleet's equity plan](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0).
5. If required, Digital Experience makes any relevant changes to the ["🗺️ Geographical factors"](https://docs.google.com/spreadsheets/d/1rCVCs-eOo-VSEG7fPLgdq5l7oSaActl5bewaWP7PnSE/edit#gid=1533353559) spreadsheet and follows through on any action items involving tax implications (i.e. registering with a new state for employer taxes).
6. If required, Digital Experience also makes changes to other core systems (e.g., creating a new email alias in Google Workspace, updating details in Carta, etc.).
7. The change is now actioned, notify the team member and close the issue.

> Note: if the Fleetie is US based and has a qualifying life event that impacts benefit coverage, they can [follow the Gusto steps](https://support.gusto.com/article/100895878100000/Change-your-benefits-with-a-qualifying-life-event) to update their coverage elections.


### Change a Fleetie's role

When Digital Experience receives [notification of a Fleetie's role changing](https://fleetdm.com/handbook/company/leadership#request-a-role-change-for-a-fleetie), The Head of Digital Experience will bring the proposed title change to the next Roundup meeting with the CEO for approval. If the proposed change is rejected, the Head of Digital Experience will inform the requesting manager as to why. If approved, use the following steps to change a Fleetie's role:  
1. Update ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0):
    - Search the spreadsheet for the Fleetie in need of a job title change.
    - Input the new job title in the Fleetie's row in the "Job title" cell.
    - Navigate to the "Org chart" tab of the spreadsheet, and verify that the Fleetie's title appears correctly in the org chart.
2. Update the departmental handbook page with the change of job title
3. [Prepare salary benchmarking information](#prepare-salary-benchmarking-information) to determine whether the teammate's current compensation aligns with the benchmarks of the new role.
   -  If the benchmark is significantly different, take the steps to [update a team member's compensation](#prepare-salary-benchmarking-information).
4. Update the relevant payroll/HRIS system.
    - For updating Gusto (US-based Fleeties):
      - Login to Gusto and navigate to "People > Team members".
      - Find the Fleetie and select them to see their profile page.
      - Under the "Compensation" heading, select edit and update the "Job title" and input the specific date the change happened. Save the changes.
    - For updating Plane (non-US Fleeties):
      - Login to Plane and navigate to "People > Team".
      - Find the Fleetie and select them to see their profile page.
      - Use the "Help" function, or email support@plane.com to notify Plane of the need to change the job title for the Fleetie. Include the Fleetie's name, current title, new title, and effective date.
      - Take any relevant steps as directed by Plane in order to make the required changes to the Fleetie's profile.


### Change a Fleetie's manager

When Digital Experience receives notification of a Fleetie's manager changing, follow these steps to ensure correct recording in our systems.
1. Update [🧑‍🚀 Fleeties](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0):
    - Search for the Fleetie's new manager, and copy the new manager's unique ID from the far left "Unique ID" column.
    - Search for the Fleetie whose manager is changing, and paste (without formatting) their new manager's unique ID in the "Reports to: (manager unique ID)" cell in the Fleetie's row.
    - Verify that the "Reports to (auto: manager name and job title)" cell in the Fleetie's row reflects the new manager's details.
    - Verify that in the new manager's row, the "# direct reports" cell reflect the correct number.
    - Navigate to the "Org chart" tab in the spreadsheet, and verify that the Fleetie now appears in the correct place in the org chart.
2. If the person's department is changing, then update both departmental handbook pages to move the person to their new department:
    - Remove the person from the "Team" section of the old department and add them to the "Team" section of the new department.
3. If the person's level of confidential access will change along with the change to their manager, then update that level of access:
    - Update Google Workspace to make sure this person lives in the correct Google Group, removing them from the old and/or adding them to the new.
    - Update 1password to remove this person from old vaults and/or add them to new vaults.
    - For a team member moving from "classified" to "confidential" access, check Gusto, Plane, and other systems to remove their access.

> **Note:** The Fleeties spreadsheet is the source of truth for who everyone's manager is and their job titles.


### Recognize employee workiversaries

At Fleet, everyone is recognized on their [workiversary](https://fleetdm.com/handbook/company/communications#workiversaries). To ensure this happens, take the following steps:

1. On the 1st of every month, use [Fleeties (private google doc)](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) to determine who is celebrating their workiversary in the next month.
2. List all team members in the ["✌️ All hands 🖐👋🤲👏🙌🤘" section of the e-group doc (confidential Google Doc)](https://docs.google.com/document/d/13fjq3T0bZGOUah9cqHVxngckv0EB2R24A3gfl5cH7eo/edit?tab=t.0#heading=h.gg4j9w7jg6g3) using the following format: `[workiversary date (DD-MMM)] - [teammate name] - [number of years at Fleet]`.
3. On the day prior to a workiversary, send the teammate’s manager a DM on Slack:


    ```
    Hey! Just a heads up, tomorrow is [teammate’s name] [number of years at Fleet] workiversary at Fleet.
    Digital Experience can post something in the #random channel to recognize them, would you like to make that post instead?
    ```
 
   > If a manager elects to post and hasn't done so by 2pm ET on the day of the workiversary, send them a friendly reminder and offer to post instead.

4. If the manager has deferred to Digital Experience, schedule a Slack post for the following day to recognize the teammate's contributions at Fleet. If you’re unsure about what to post, take a look at what’s been [posted previously](https://docs.google.com/document/d/1Va4TYAs9Tb0soDQPeoeMr-qHxk0Xrlf-DUlBe4jn29Q/edit).


### Prepare salary benchmarking information

1. Use the relevant template text in the README section of the [¶¶ 💌 Compensation decisions document](https://docs.google.com/document/d/1NQ-IjcOTbyFluCWqsFLMfP4SvnopoXDcX0civ-STS5c/edit?usp=sharing) for a current Fleetie, a new role, a prospective hire, or other benchmarking use case.
2. Copy the template text and paste at the end of the document.
3. Fill in details as required, pulling from [🧑‍🚀 Fleeties spreadsheet](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) and [equity spreadsheet](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit?usp=sharing) as required.
4. Use the teammate's information to benchmark in [Pave](https://www.pave.com/) (login details in 1Password). You can pattern match from previous benchmarking entries and include all company assumptions. Add the direct link to the Pave benchmark.


### Update a team member's compensation

To [change a teammate's compensation](https://fleetdm.com/handbook/company/communications#compensation-changes), follow these steps:
1. Create a copy of the ["Values assessment" template](https://docs.google.com/spreadsheets/d/1P5TyRV2v-YN0aR_X8vd8GksKcr3uHfUDdshqpVzamV8/edit?usp=drive_link) and move it to the teammate's [personnel folder in Google Drive](https://drive.google.com/drive/folders/1UL7o3BzkTKnpvIS4hm_RtbOilSABo3oG?usp=drive_link).
2. Share the values assessment document with the manager via Slack DM (include the Head of Digital Experience) and ask the manager to fill out the values assessment.
3. Once the values assessment is complete, [prepare salary benchmarking information](#prepare-salary-benchmarking-information) and at-mention the Head of Digital Experience in the workiversary issue. Add a "DISCUSS" item to the roundup doc so the compensation change can be reviewed with the CEO.
4. Once compensation decisions have been finalized, the Head of Digital Experience will send the teammate's manager a Slack DM to communicate the compensation decision and asking them to inform the teammate.
5. Update the respective payroll platform (Gusto or Plane) by navigating to the personnel page, selecting salary field, and updating with an effective date that makes the next payroll.
6. Update the [equity spreadsheet](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit?usp=sharing) (internal doc) by copying Adding to the "Notes" cell,
  - Update the "¶¶ Annual OTE ($)" column with the new compensation information.
  - Update the "Last compensation change" column with the effective date from payroll platform.
  - Update the "¶¶ Notes" column. **⚠️ MAKE SURE NOT TO DELETE ANY EXISTING NOTES ⚠️** Add the note to the top of the cell using the following format: `As of YYYY-MM-DD OTE +15k` (pattern match off of other 2024 notes). Link your note to the relevant title in the ["¶¶ 💌 Compensation decisions (offer math)" (classified Google Doc)](https://docs.google.com/document/d/1NQ-IjcOTbyFluCWqsFLMfP4SvnopoXDcX0civ-STS5c/edit?tab=t.0#heading=h.slomq4whmyas).
  - If the company decides on an additional equity grant as part of a compensation change, note the previous equity and new situation in detail in the "Notes" column of the equity plan. Update the "Grant started?" column to "todo" which adds it to the queue for the next time grants are processed (quarterly).
7. Calculate the monthly burn rate increase percentage and notify the CEO via a Slack DM.


### Grant role-specific license to a team member

Certain new team members, especially in go-to-market (GTM) roles, will need paid access to paid tools like Salesforce and LinkedIn Sales Navigator immediately on their first day with the company. Gong licenses that other departments need may [request them from Digital Experience](https://fleetdm.com/handbook/digital-experience#contact-us) and we will make sure there is no license redundancy in that department.


### Process a tool upgrade request from a team member

- A Fleetie may request an upgraded license seat for Fleet tools by submitting an issue through GitHub.
- Digital Experience will upgrade or add the license seat as needed and let the requesting team member know they did it.


### Downgrade an unused license seat

- On the first Wednesday of every quarter, the CEO and Head of Digital experience will meet for 30 minutes to audit license seats in Figma, Slack, GitHub, Salesforce and other tools.
- During this meeting, as many seats will be downgraded as possible. When doubt exists, downgrade.
- Afterward, post in #random letting folks know that the quarterly tool reconciliation and seat clearing is complete, and that any members who lost access to anything they still need can submit a GitHub issue to Digital Experience to have their access restored.
- The goal is to build deep, integrated knowledge of tool usage across Fleet and cut costs whenever possible. It will also force conversations on redundancies and decisions that aren't helping the business that otherwise might not be looked at a second time.  


### Add a seat to Salesforce

Here are the steps we take to grant appropriate Salesforce licenses to a new hire:
- Go to ["My Account"](https://fleetdm.lightning.force.com/lightning/n/standard-OnlineSalesHome).
- View contracts -> pick current contract.
- Add the desired number of licenses.
- Sign DocuSign sent to the email.
- The order will be processed in ~30m.
- Once the basic license has been added, you can create a new user using the new team member's `@fleetdm.com` email and assign a license to it.
  - To enable email sync for a user:
    - Navigate to the [user’s record](https://fleetdm.lightning.force.com/lightning/setup/ManageUsers/home) and scroll to the bottom of the permission set section.
    - Add the “Inbox with Einstein Activity Capture” permission set and save.
    - Navigate to the ["Einstein Activity Capture Settings"](https://fleetdm.lightning.force.com/lightning/setup/ActivitySyncEngineSettingsMain/home) and click the "Configurations" tab.
    - Select "Edit", under "User and Profile Assignments" move the new user's name from "Available" to "Selected", scroll all the way down and click save.


### Inform managers about hours worked

Every Friday, we collect hours worked for all hourly employees at Fleet, including core team members and consultants, regardless of their location. Consultants submit their hours through Gusto (US consultants) or Plane.com (international consultants) for DRI (generally their manager) review. Here's how:
1. Find the DRI using the [Digital Experience KPIs](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0).
2. Copy the template in the consultants column of the KPIs and send the teammate's DRI a direct message in Slack hours with an FYI including the total hours logged since last Saturday at midnight. For international teammates, they cannot enter hours weekly in Plane.com, so you will need to request the hours worked from them in order to have the DRI approve them.
3. The following Monday, check for updates to logged hours and ensure the KPI sheet aligns with HRIS records. If there are discrepancies between what was previously reported, reconfirm logged hours with the teammate's DRI and update the KPI sheet to reflect the correct amount.


### Change the DRI of a consultant

1. In the [KPIs](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0) sheet, find the consultant's column.
2. Change the DRI documented there to the new DRI who will receive information about the consultant's hours.


## Add an advisor

First: Advisor agreements are sent through [DocuSign](https://www.docusign.com/), using the "Advisor Agreement" template.
- Update the ["Advisors" sheet](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit#gid=1803674483)
  >*Be sure to mark any columns that haven't been completed yet as "TODO"*
- Update the "Equity plan" sheet (which should have been automatically updated after updating "Advisors" thanks to the global unique IDs next to each row which are used to connect the spreadsheets) to reflect the default number of shares for advisor equity grants.
- Send the advisor agreement [through Docusign](https://apps.docusign.com/send/templates?view=shared&folder=0482b0fd-a752-41be-93a0-185e2fb7ef54) using the CEO's account, pulling the advisor's email address from a recent calendar event on the CEO's calendar.
- Complete the first step of signing, which involves filling in the number of shares.
- Then wait for the advisor to sign.  (Fleet's CEO will sign after that.)

Then, to finalize a new advisor after signing is complete:
- Schedule quarterly recurring 1h meeting between the CEO and the advisor, with 30m of recurring prep scheduled back to back ahead of the meeting.
- Update the status columns in the ["Advisors" sheet](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit#gid=1803674483) to show that the agreement has been signed, and ask the new advisor to add us on [LinkedIn](https://www.linkedin.com/company/71111416), [Crunchbase](https://www.crunchbase.com/organization/fleet-device-management), and [Angellist](https://angel.co/company/fleetdm).
- Update "Equity plan" status columns to reflect updated status for this advisor, and to ensure the advisor's equity is queued up for the next quarterly equity grant ritual.


### Approve a new position

When review is requested on a proposal to open a new position, the Head of Digital Experience will complete the following steps when reviewing the pull request:
1. Confirm the new row in "Fleeties" has a manager, job title, and department, that it doesn't have any corrupted spreadsheet formulas or formatting, and that the start date is set to the first Monday of the next month.
2. Confirm the job description consists only of changes to "Responsibilities" and "Experience," with an appropriate filename, and that the content looks accurate, is grammatically correct, and is otherwise ready to post in a public job description on fleetdm.com.
3. Ballpark and document compensation research for the role based on 
   - _Add screenshot:_ Scroll to the very bottom of ["¶¶ 💌 Compensation decisions (offer math)"](https://docs.google.com/document/d/1NQ-IjcOTbyFluCWqsFLMfP4SvnopoXDcX0civ-STS5c/edit#heading=h.slomq4whmyas) and add a new heading for the role, pattern-matching off of the names of other nearby role headings. Then create written documentation of your research for future reference.  The easiest way to do this is to take screenshots of the [relevant benchmarks in Pave](https://pave.com) and paste those screenshots under the new heading.
4. Decide whether to approve this role or to consider it a different time.  If approving, then:
   - _Update financial model:_ Update ["¶ Financial model"](https://docs.google.com/spreadsheets/d/1tIcuwhmOKolnwNJqQ0zH5rWCqjawYzySDsKTb98RvxI/edit?gid=1184088923#gid=1184088923)
   - _Update team database:_ Update the row in ["¶¶ 🥧 Equity plan"](https://docs.google.com/spreadsheets/d/1_GJlqnWWIQBiZFOoyl9YbTr72bg5qdSSp4O3kuKm1Jc/edit#gid=0) using the benchmarked compensation and share count.
     - _Salary:_ Enter the salary: If the role has variable compensation, use the role's OTE (on-target earning estimate) as the budgeted salary amount, and leave a note in the "Notes (¶¶)" cell clarifying the role's bonus or commission structure.
     - _Equity:_ Enter the equity as a number of shares, watching the percentage that is automatically calculated in the next cell.  Keep guessing different numbers of shares until you get the derived percentage looking like what you want to see.
   - _Create Slack channel:_ Create a private "#YYYY-hiring-xxxxxx" Slack channel (where "xxxxxx" is the job title and YYYY is the current year) for discussion and invite the hiring manager and Head of Digital Experience.
   - _Publish opening:_ Approve and merge the pull request.  The job posting will go live within ≤10 minutes.
   - _Track as approved in "Fleeties":_ In the "Fleeties" spreadsheet, find the row for the new position and update the "Job description" column and replace the URL of the pull request that originally proposed this new position with the URL of the GitHub merge commit when that PR was merged.
   - _Reply to requestor:_ Post a comment on the pull request, being sure to include a direct link to their live job description on fleetdm.com.  (This is the URL where candidates can go to read about the job and apply.  For example: `fleetdm.com/handbook/company/product-designer`):
     ```
     The new opening is now live!  Candidates can apply at fleetdm.com/handbook/company/railway-conductor.
     ```

> _Most columns of the "Equity plan" are updated automatically when "Fleeties" is, based on the unique identifier of each row, like `🧑‍🚀890`.  (Advisors have their own flavor of unique IDs, such as `🦉755`, which are defined in ["Advisors and investors"](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit).)_



### Convert a Fleetie to a consultant

If a Fleetie decides they want to move to being a [consultant](https://fleetdm.com/handbook/company/leadership#consultants), either the Fleetie or their manager need to create a [custom issue for the Digital Experience team](https://github.com/fleetdm/confidential/issues/new?assignees=&labels=%23g-digital-experience&projects=&template=1-custom-request.md&title=Request%3A+_______________________) to notify them of the change.
Once notified, Digital Experience takes the following steps:
1. Confirm the following details with the Fleetie:
    - Date of change
    - Term of consultancy (time period)
    - Hours/capacity expected (hours per week or month)
    - Confirm hourly rate
2. Once details are confirmed, use the information given to create the consulting agreement for the Fleetie (either in docusign (US-based) or via Plane (international)), and send to their personal email for signature. Once signed, save in Fleetie's [employee file](https://drive.google.com/drive/folders/1UL7o3BzkTKnpvIS4hm_RtbOilSABo3oG?usp=drive_link).
3. Schedule the Fleetie's final day in HRIS (Gusto or Plane).
4. Update final day in ["🧑‍🚀 Fleeties"](https://docs.google.com/spreadsheets/d/1OSLn-ZCbGSjPusHPiR5dwQhheH1K8-xqyZdsOe9y7qc/edit#gid=0) spreadsheet.
5. Create an [offboarding issue](https://github.com/fleetdm/classified/blob/main/.github/ISSUE_TEMPLATE/%F0%9F%9A%AA-offboarding-____________.md) for the Fleetie converting to a consultant, and confirm with their manager if there is a need to retain any tools or access while they are a consultant (default to removing all access from Fleet email, and migrating to personal email for Slack and other tools unless there is a business case to retain the Fleet email and associated tool access).
6. Follow the offboarding issue for next steps, including communicating to teammates and updating equity plan.


### Change the "Integrations admin" Salesforce account password

Salesforce requires that the password to the "Integrations admin" account is changed every 90 days. When this happens, the Salesforce integrations on the Fleet website/Hydroplane will fail with an `INVALID_LOGIN` error. To prevent this from happening, a member of the Digital expererience team will:

1. Log into the "Integrations admin" account in Salesforce.
2. Change the password and save it in the shared 1Password vault.
3. Request a new security token for the "Integrations admin" account by clicking the profile picture » `Settings` » `Reset my security token` (This will be sent to the email address associated with the account).
4. Update the `sails_config__custom_salesforceIntegrationPasskey` config variable in Heroku to be `[password][security token]` (For both the Fleet website and Hydroplane).


### Review Fleet's US company benefits

Annually, around mid-year, Fleet will be prompted by Gusto to review company benefits. The goal is to keep changes minimal. Follow these steps:
1. Log in to your [Gusto admin account](https://gusto.com/).
2. Navigate to "Benefits" and select "Renewal survey".
3. Complete the survey questions, aiming for minimal changes.
4. Approximately 2-3 months after survery completion, Gusto will suggest plans based on Fleet's responses. Choose plans with minimal changes.
5. Gusto will offer these plans to employees during open enrollment, with new coverage starting 3-4 weeks afterward.
   

### Prepare for the All hands

- **Every month** the Apprentice will do the prep work for the monthly "✌️ All hands 🖐👋🤲👏🙌🤘" call.
  -  In the ["👋 All hands" folder](https://drive.google.com/drive/folders/1cw_lL3_Xu9ZOXKGPghh8F4tc0ND9kQeY?usp=sharing), create a new folder using "yyyy-mm - All hands".
  - Update "End of the quarter" slides to reflect the current countdown.
  - Download a copy of the previous month's keynote file and rename the copy pattern matching existing files.
  - Update the slides to reflect the current "All hands" date (e.g. cover slides month and the "You are here" slide)'
  - Update slides that contain metrics to reflect current information using the [📈 KPIs](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit#gid=0) doc.
  - Update the "Spotlight slide" for guest speakers.
  - Add new customer logos from Mike's bookmarks ["Customers list"](https://fleetdm.lightning.force.com/lightning/o/Account/list?filterName=00B4x00000CTHP8EAP) and Google "Company name" to find the current logo.

- **First "All hands" of the quarter**
  - Audit the "Strategy" slide.
  - Audit the "Goals" slide

The day before the All hands, Mike will prepare slides that reflect the CEO vision and focus. 


### Share recording of all hands meeting

The Apprentice will post a link to the All hands Gong recording and slide deck in Slack.
Template to use:

```
Thanks to everyone who contributed to today's "All hands" call.

:tv: If you weren't able to attend, please *[watch the recording](Current-link-to-Gong-recording)* _(1.5x playback supported)_.

You can also grab a copy of the [original slides](https://fleetdm.com/handbook/company/communications#all-hands) for use in your own confidential presentations.
```

1. Copy and paste the template to the "[# general](https://fleetdm.slack.com/archives/C019FNQPA23)" Slack channel.
2. Open [Gong recording](https://us-65885.app.gong.io/home?workspace-id=9148397688380544352&r=m) and click `Share call`, then click `Share with customers`, then `Copy link`.
3. Paste the url `*[Watch the recording](`here-in-your-template-message`)*`.

<img width="464" alt="image" src="https://github.com/Sampfluger88/fleet/assets/108141731/c2002cfa-a0f6-4349-bb06-71104f6cdce1">

4. Schedule the Slack message to go out at 6pm CT (18:00).


### Process and backup Sid agenda

Every two weeks, our CEO Mike has a meeting with Sid Sijbrandij. The CEO uses dedicated (blocked, recurring) time to prepare for this meeting earlier in the week.
1. 30 minutes After each meeting [archive the "💻 Sid : Mike(Fleet)" agenda](https://fleetdm.com/handbook/digital-experience#archive-a-document), moving it to the [(¶¶) Sid archive](https://drive.google.com/drive/folders/1izVfIBt2nr4APlkm36E6DJg1k1PDjmae) folder in Google Drive.
2. **In the backup copy**, create Google Doc comments assigning all Fleet TODOs to the correct DRI.   
3. In the ¶¶¶¶🦿🌪️CEO Roundup doc, update the URL in `Sam: FYI: Agenda from last time:` [LINK](link).


### Process and backup E-group agenda 

Follow these steps to process and backup the E-group agenda: 
1. [Archive the E-group agenda](https://fleetdm.com/handbook/digital-experience#archive-a-document) after each meeting, moving it to the ["¶¶ E-group archive"](https://drive.google.com/drive/u/0/folders/1IsSGMgbt4pDcP8gSnLj8Z8NGY7_6UTt6) folder in Google Drive.
2. **In the backup copy**, leave Google Doc comments assigning all TODOs to the correct DRI.  
3. If the "All hands" meeting has happened today remove any spotlights covered in the current "All hands" presentation.


### Process the help-being-ceo Slack channel

The Apprentice will perform the following steps to process all communication from the CEO in the [help-being-ceo Slack channel](https://fleetdm.slack.com/archives/C03U703J0G5).
1. As soon as the message is received in the channel, add the "`:eyes:` (👀)" emoji to the Slack message to signify that you have seen and understood the task or question.
2. Start a Slack thread to add any context or let the stakeholders know the status of the task. 
3. After each task is completed, apply the "`:white_check_mark:`" (✅) to the slack message.


### Unroll a Slack thread

From time to time the CEO will ask the Apprentice to unroll a Slack thread into a well-named whiteboard Google doc for safekeeping and future searching. 
  1. Start with a new doc.
  2. Name the file with "yyyy-mm-dd - topic" (something empathetic and easy to find).
  3. Use CMD+SHFT+V to paste the Slack convo into the doc.
  4. Reapply formatting manually (be mindful of quotes, links, and images).
      - To copy images right-click+copy and then paste in the doc (some resizing may be necessary to fit the page).


### Delete an accidental meeting recording

It's not enough to just "delete" a recording of a meeting in Gong.  Instead, use these steps:

- Wait for at least 30 minutes after the meeting has ended to ensure the recording and transcript exist and can be deleted.
- [Sign in to Gong](https://us-65885.app.gong.io/deals?company-id=2676443513846037003&workspace-id=9148397688380544352&board-id=8761946992754097113&view-mode=DEALS&tab-idx=0&account-activity=true&owner-ids=&owner-team-ids=5778354842532790437&timespan-id=34&sort-by=DealActivity&sort-field=%7B%22type%22%3A%22RegularField%22%2C%22name%22%3A%22DealActivity%22%7D&sort-order=DESC&owner-id=5778354842532790437&include-team=true) through the CEO's browser.
- Scroll down to `Conversations`
- Select the call recording no longer needed
- Click the "hotdog" menu in the right-hand corner
<img width="264" alt="image" src="https://github.com/fleetdm/fleet/assets/108141731/86948d02-a972-42ef-9a2d-1d93f24a1780">
- `Delete recording`
- Search for the title of the meeting Google Drive and delete the auto-generated Google Doc containing the transcript. 
- Always check back to ensure the recording **and** transcript were both deleted.


### Communicate Fleet's potential energy to stakeholders

On the first business day of every month, the Head of Digital Experience will send an update to the stakeholders of Fleet using the following steps:
1. Navigate to the "[🪴🌧️🦉 Investor updates](https://docs.google.com/spreadsheets/d/10T7Q9iuHA4vpfV7qZCm6oMd5U1bLftBSobYD0RR8RkM/edit?gid=0#gid=0)" spreadsheet and confirm the data in each column matches the header of that column (e.g. the "Headcount" column actually has headcount values in it). Do this by confirming the "Remote column" value corresponds to the correct column "letter" in the "Weekly updates" tab of the "[📈 OKRs (quarterly goals) + KPIs (everyday metrics)](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit?gid=0#gid=0)" spreadsheet.
2. Confirm KPI's are up-to-date. If any KPI's aren't completed, at mention the e-group member responsible and ask that the KPI's be completed ASAP in order to send the investor update. 
3. Copy the following template into an outgoing email with the subject line: "[Investor update] Fleet, YYYY-MM".

```
Hi investors and friends,


FYI we just updated the self-service investor update portal with the numbers from last month:  https://docs.google.com/spreadsheets/d/10T7Q9iuHA4vpfV7qZCm6oMd5U1bLftBSobYD0RR8RkM/edit#gid=0


Thanks for your support,
Mike and the Fleet team

```

4. Address the email to the executive team's Gmail.
5. Using the [🌧️🦉 Investors + advisors](https://docs.google.com/spreadsheets/d/15knBE2-PrQ1Ad-QcIk0mxCN-xFsATKK9hcifqrm0qFQ/edit#gid=1068113636) spreadsheet, bcc the correct individuals and send the email.


### Archive a document

Follow these steps to archive any document:
1. Create a copy of the document prefixed with the date using the format "`YYYY-MM-DD` Backup of `DOCUMENT_NAME`" (e.g. "2024-03-22 Backup of 🪂🗞️ Customer voice").
2. Be sure to "Share it with the same people", "Copy comments and suggestions", and "Include resolved comments and suggestions" as shown below.

<img width="455" alt="Screenshot 2024-03-23 at 12 14 00 PM" src="https://github.com/fleetdm/fleet/assets/108141731/1c773069-11a7-4ef4-ab43-8f7c626e4b10">

3. Save this backup copy to the same location in Google Drive where the original is found.
4. Link to the backup copy at the top of the original document. Be sure to use the full URL, no abbreviated pill links (e.g. "Notes from last time: URL_OF_MOST_RECENT_BACKUP_DOCUMENT").
5. Delete all non-structural content from the original document, including past meeting notes and current answers to "evergreen" questions.


### Process the CEO's inbox

- The Apprentice is [responsible](https://fleetdm.com/handbook/company/why-this-way#why-direct-responsibility) for [processing all email traffic](https://docs.google.com/document/d/1gH3IRRgptrqSYzBFy-77g98JROTL8wqrazJIMkp-Gb4/edit#heading=h.i7mkhr6m123r) prior to CEO review to reduce the scope of Mike's inbox to only include necessary and actionable communication.
 -  Marking spam emails as read (same for emails Mike doesn't actually need to read).
 -  Escalate actionable sales communication and update Mike directly.
 -  Ensure all calendar invites have the necessary documents included.
 -  Forward any emails from customers about paying Fleet to the Buisness Operations department using [Fleet's billing email](https://docs.google.com/document/d/1tE-NpNfw1icmU2MjYuBRib0VWBPVAdmq4NiCrpuI0F0/edit#heading=h.wqalwz1je6rq).


### Process the CEO's calendar

Time management for the CEO is essential.  The Apprentice processes the CEO's calendar multiple times per day.

- **Clear any unexpected new events or double-bookings.** Look for any new double-bookings, invites that haven't been accepted, or other events you don't recognize.
  1. Double-book temporarily with a "UNCONFIRMED" calendar block so that the CEO ignores it and doesn't spend time trying to figure out what it is.
  2. Go to the organizer (or nearest fleetie who's not the CEO):
    - Get full context on what the CEO should know as to the purpose of the meeting and why the organizer thinks it is helpful or necessary for the CEO to attend.
    - Remind the organizer with [this link to the handbook that all CEO events have times chosen by Savannah before booking](https://fleetdm.com/handbook/company/communications#schedule-time-with-the-ceo).
  3. Bring prepped discussion item about this proposed event to the next CEO roundup, including the purpose of the event and why it is helpful or necessary for the CEO to attend (according to the person requesting the CEO's attendance).  The CEO will decide whether to attend.
  4. Delete the "UNCONFIRMED" block if the meeting is confirmed, or otherwise work with the organizer to pick a new time or let them know the decision.

- **Prepare the agenda for any newly-added meetings**: [Meeting agenda prep](https://docs.google.com/document/d/1gH3IRRgptrqSYzBFy-77g98JROTL8wqrazJIMkp-Gb4/edit#heading=h.i7mkhr6m123r) is especially important to help the CEO focus and transition quickly in and between meetings. Using the CEO's browser, prepare each document by including the following:

> If a meeting agenda has to be created from scratch, be sure to move it to the "Meeting notes" folder in Google Drive so that he isn't locked out of any documents.
> If preparing for a meeting with a current advisor, use the existing journal as the meeting agenda using these steps:
> 1. Search for the journal in Mike's browser using the advisor's name or email.
> 2. Update the journal by adding the date of the meeting as an H3 in the Google document (pattern matching the document) and link the document to the calendar description.
 
  1. LinkedIn profile url of all outside participants. Connect with any of the attendees that the CEO is not already connected to on LinkedIn, this should always be a blank connect request meaning "Send without note". Nest everything from prep under the LinkedIn url (ie all under #1)
  2. A screen-shot of LinkedIn profile pic
  3. Company name (in doc title, file name and Google calendar event title)
  4. Correct date (20XX-XX-XX in doc title and file name)
  5. Context that helps the CEO to understand the purpose of the meeting at a glance from:
    - CEO's email
    - LinkedIn messages (careful not to mark things as read!)
    - Google Drive 
  6. Edit the calendar event description, changing “Notes” to “Agenda” when you're finished preparing the document to signify that this meeting has been prepped.


### Check LinkedIn for new activity 

Once a day the Apprentice will check LinkedIn for unread messages and pending connect request. 

  1. Log into the CEO's [LinkedIn](https://www.linkedin.com/search/results/all/?sid=s2%3A) and bring up the messaging window.
  2. Filter out all read messages by clicking "filter" and then "Unread".
  3. Bring all unreads to the CEO during the daily roundup.
     
To check for pending connect requests, perform the following steps:
  1. Log into the CEO's LinkedIn (if you're not already) and click "My Network".
  2. Bring all pending connect requests to the CEO during the daily roundup.



### Add LinkedIn connections to CRM

To add the most recent connections from Linkedin to our CRM, follow these steps:
  1. Log into the CEO's LinkedIn (if you're not already) and click "My Network", then "Connections" and open each person's LinkedIn page in a new tab.
  2. Log into our CRM using the Fleet's billing login (in 1Password) in another tab.
  3. Scroll down to the "Experience" section to find the person's current employer and search for that account in the CRM database. 
  4. In LinkedIn, navigate to the employer company profile. Click "insights" to see how many employees are listed and update the "Employees" field on the CRM account. 
  5. The "Account rating" on the CRM account must be a 🦄, if they're not, Do not add the contact. Move on to the following person.
  6. If the account is a 🦄, click "All contacts" and make sure they're not a contact already.
  7. Create a new contact on the account: Click "New", fill out their full name, title, role, buying situation, and LinkedIn URL, then save the record.
  8. Click on the new contact (their name) that you created and change their psychological stage to "intrigued" (we consider them intrigued since they've reached out to the CEO via LinkedIn).


### Connect with active community members

Once a week, the Apprentice will review the "community activity report" and add the LinkedIn URLs into the campaign. This will send out a connection request to those who liked, shared, commented, etc. a post on LinkedIn. 
 1. Export the [community activity report](https://fleetdm.lightning.force.com/lightning/r/Report/00OUG000002j3wf2AA/view).
 2. Copy the LinkedIn URLs.
 3. Paste the LinkedIn URLs in the [appropriate Dripify campaign](https://app.dripify.io/campaigns/1291030).

 ![image](https://github.com/user-attachments/assets/dc20c4c2-9691-4e70-bb9c-90b725403571)



### Schedule travel for the CEO

The Apprentice will verify daily that the CEO's calendar is accurate for any commitment or booked travel. The Apprentice is the DRI for scheduling all travel arrangements for the CEO, including flights, hotels, and reservations if needed. The CEO's traveling preferences in descending order of importance are:
  - Direct flight whenever possible  (as long as the cost of the direct flight is ≤2x the cost of a reasonable non-direct flight)
  - Select a non-middle seat, whenever possible
  - Don't upgrade seats (unless there's a cheap upgrade that gets a non-middle seat, or if a flight is longer than 5 hours.  Even then, never buy a seat upgrade that costs >$100.)
  - The CEO does not like to be called "Michael".  Unfortunately, this is necessary when booking flights.  (He has missed flights before by not doing this.)
  - Default to carry-on only, no checked bags.  (For trips longer than 5 nights, add 1 checked bag.)
  - Use the Brex card.
  - Frequent flyer details of all (previously flown) airlines are in 1Password as well as important travel documents.
  - The CEO will schedule his own transportation (e.g. to/from hotel/event location) while traveling. If the CEO is traveling to an event or meeting where other Fleeties are present, he may use travel time (e.g. an Uber ride) as time to align with other team members in person.


### Schedule CEO interview

Use the following steps to schedule an interview between a candidate and the CEO:
1. Once you receive a [CEO interview request](https://fleetdm.com/handbook/company/leadership#hiring-a-new-team-member), apply the "eyes" (👀) emoji to the Slack post to acknowledge you've seen the request.
2. Reach out to the candidate via email to find a time when the CEO and candidate are both available.
   > This entire process takes an hour for the CEO: a 30-minute interview followed by a 30-minute "¶¶ Postgame" Be sure to offer times that accommodate this.
3. [Make a copy of the "¶¶ CEO interview template"](https://docs.google.com/document/d/1yARlH6iZY-cP9cQbmL3z6TbMy-Ii7lO64RbuolpWQzI/copy) (private Google doc) and move it to the "[¶¶ Interview feedback](https://drive.google.com/drive/folders/1v5Z1WB9S855hLZMUWgOiXA_ei2EpEGlA?usp=drive_link)" folder in Google Drive. 
4. Prep the CEO interview doc:
   - Change file name and heading of doc to `¶¶ CANDIDATE_NAME (CANDIDATE_TITLE) <> Mike McNeil, CEO final interview (YYYY-MM-DD)`.
   - Add candidate's personal email in the "👥" (attendees) section at the top of the doc.
   - Add candidate's [LinkedIn url](https://www.linkedin.com/search/results/all/?keywords=people) on the first bullet for Mike.
   - Share the CEO interview doc with the hiring manager as a "Commenter".
5. Link the CEO interview doc at the top of the "feedback" doc shared in the CEO interview request
6. Create a Google Calendar event at a time when the CEO and the candidate are both available.
   - Create a Google Calendar event matching the title of the interview doc.
   - Add the interview doc to the calendar event description as the agenda (i.e. `Agenda: INTERVIEW_DOC_FULL_URL`) and save the calendar event.
7. Schedule a 30-minute "¶¶ Postgame" working session for the CEO to evaluate the candidate and give his recommendation.
8. In the hiring channel for the position, apply the "green-check-mark" (✅) emoji to the CEO interview request to confirm the request has been processed. 


### Program the CEO to do something

1. If necessary or if unsure, immediately direct message the CEO on Slack to clarify priority level, timing, and level of effort.  (For example, whether to schedule 30m or 60m to complete in full, or 30m planning as an iterative step.)
2. If there is not room on the calendar to schedule this soon enough with both Mike and Sam as needed (erring on the side of sooner), then either immediately direct message the CEO with a backup plan, or if it can obviously wait, then discuss at the next roundup.
3. Create a calendar event with a Zoom meeting for the CEO and Apprentice.  Keep the title short.  For the description, keep it very brief and use this template:

```
Agenda:
1. Apprentice: Is there enough context for you (CEO) to accomplish this?
2. Apprentice: Is this still a priority for you (CEO) to do.. right now?  Or should it be "someday/maybe"?
3. Apprentice: Is there enough time for you (CEO) to do this live? (Right now during this meeting?)
4. Apprentice: What are the next steps after you (CEO) complete this?
5. Apprentice: LINK_TO_DOC_OR_ISSUE
```


### Confirm CEO shadow dates

Use the following steps to confirm CEO shadow dates:
1. Create an "All day", "Free" event on the CEO's calendar that matches the CEO shadow dates and name the calendar event "CEO shadow - [NAME] (Job title)".
2. Go through the calendar and make sure all private meetings (e.g. 1:1's, E-Group, and quarterly board meetings) have "[no shadows]" in the event title.
3. Add a "DISCUSS: CEO shadow YYYY-MM-DD to YYYY-MM-DD TEAM_MEMBER_NAME - POSITION" item to the meeting agenda
4. Attend the next "🐈‍⬛🌪️ Roundup (~ceo)" meeting to make the CEO aware of the dates and confirm the "shadowability" of external and nonrecurring internal meetings.

> After the team member notifies the Head of Digital Experience (via Slack), the Head of DigExp will bring the dates to the next "🐈‍⬛🌪️ Roundup (~ceo)".


### Monitor compliance tests

1. Every Monday, log in to Vanta and create GitHub issues for any tests that are due or need remediation in the next 3 weeks.    
2. To do this, access "Tests" on the left side menu.  This will provide a status report of the tests, when they are due, and who the DRI is.  
3. Click on a test, then click on "Tasks".  
4. Click on "Create task." Then, "Create GitHub issue."
5. This will bring you to a screen where you can select the appropriate DRIs and GitHub labels (multiple, if necessary, but always include the "#g-digital-experience" label). Vanta will autopopulate the issue with a brief description of the test due and what needs to be remediated. You can manually add details if necessary.
6. Follow up with the DRI of each issue daily until it's resolved. As needed, loop in their manager, Fleet's CTO, or the Head of Digital Experience. If the test is within 3 days of being overdue, DM the fleetie and their manager, asking to have the issue prioritized and completed before the due date. 


## Rituals

- Note: Some rituals (⏰) are especially time-sensitive and require attention multiple times (3+) per day.  Set reminders for the following times (CT):
  - 9:30 AM _(/before first meeting)_
  - 12:30 PM CT _(/beginning of "reserved block")_
  - 6:30 PM CT _(/after last meeting, before roundup / Japan calls)_

<rituals :rituals="rituals['handbook/digital-experience/digital-experience.rituals.yml']"></rituals>


#### Stubs
The following stubs are included only to make links backward compatible.



<meta name="maintainedBy" value="Sampfluger88">
<meta name="title" value="🌐 Digital Experience">

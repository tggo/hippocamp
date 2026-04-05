# CRM Hygiene and Process Notes

## Salesforce Instance

- **Instance**: Nimbus Software Salesforce (Enterprise Edition)
- **Admin**: Taylor Kim (RevOps Manager)
- **URL**: nimbus.my.salesforce.com
- **Integration**: Connected to Gong for call recording, Outreach for sequencing, and Clari for forecasting

## Required Fields — Opportunity Object

Every opportunity must have the following fields populated before it can advance to the next stage. Taylor runs a weekly data quality report and flags non-compliant records.

### At Discovery Stage
- **MEDDIC Score**: Composite 0-30 score (5 points per MEDDIC element). Minimum 10 to advance.
- **Champion Name**: Free text, must match a Contact record
- **Pain Point Summary**: 2-3 sentences describing the business problem
- **Source**: Lead source (Inbound, Outbound SDR, Referral, Event, Partner)
- **Competitor**: Multi-select picklist (DataFlow, SyncWave, In-house, None identified)

### At Proposal Stage
- **Economic Buyer**: Must be a Contact record with title populated
- **Decision Criteria**: Free text, minimum 3 criteria listed
- **Expected Close Date**: Must be within 90 days
- **ARR Amount**: Calculated field based on tier and seat count
- **Next Step**: Required free text, updated after every customer interaction

### At Negotiation Stage
- **Legal Contact**: Procurement or legal contact at customer
- **Discount Approval**: If any discount above 10%, requires VP Sales (Jennifer Torres) approval via Salesforce approval workflow
- **Contract Term**: Picklist (Annual, 2-year, 3-year)
- **Payment Terms**: Net 30, Net 45, or Net 60

## Weekly Pipeline Review

Every Monday at 10:00 AM PT, Jennifer Torres conducts a 60-minute pipeline review via Zoom. Format:

1. **Deal-by-deal review** (40 min): Each AE presents their top 3 deals with updates since last week. Use the Clari forecast view as the source of truth. Jennifer will challenge any deal that has been in the same stage for more than 3 weeks without a documented next step.

2. **Forecast call** (10 min): Each AE provides their commit, best case, and upside numbers. The team forecast rolls up to Jennifer who reports to CEO Laura Chen every Tuesday.

3. **Pipeline generation review** (10 min): SDR metrics from Outreach (emails sent, reply rates, meetings booked). Jordan and Samantha present their top prospect accounts for the week.

## Deal Desk Process

For non-standard deals (custom pricing, multi-year with annual escalators, professional services bundles), submit a Deal Desk request to Taylor Kim via the Salesforce Deal Desk object. SLA is 48 hours for pricing approval. For deals over $200K ARR, Mark Davidson (CFO) must co-approve.

## Data Cleanup Rules

- Close-lost deals must have a loss reason selected (Lost to Competitor, Lost to No Decision, Lost to Budget, Lost to Timing)
- Stale opportunities (no activity for 30+ days) are auto-flagged by a Salesforce flow and assigned to the AE's manager for review
- Contact records must have a valid email and at least one phone number
- All calls over 5 minutes must be logged in Gong and linked to the opportunity

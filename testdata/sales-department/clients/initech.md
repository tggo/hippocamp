# Initech — New Prospect

## Account Summary

- **Company**: Initech (enterprise software company)
- **Industry**: Technology / SaaS
- **Headquarters**: San Jose, CA
- **Employees**: 1,800
- **Annual Revenue**: $220M
- **Current CloudSync Pro ARR**: $0 (new prospect)
- **Estimated deal size**: $120,000 ARR (Enterprise tier, 250 seats)
- **Account Executive**: Marcus Johnson
- **SDR**: Jordan Lee (sourced via outbound LinkedIn campaign)

## Key Contacts

- **Michael Chen** — Head of IT, primary contact and internal champion. Michael attended our webinar on "Breaking Down Data Silos in SaaS Companies" on February 20 and requested a demo. He has budget authority up to $150K and reports directly to the CIO (Karen Walsh). Email: m.chen@initech.com, Phone: (408) 555-0317.
- **Deepa Krishnamurthy** — Director of Engineering, technical evaluator. Deepa will lead the technical proof-of-concept and has requested access to our API documentation and a sandbox environment. She is particularly interested in our conflict resolution engine for bi-directional Salesforce-HubSpot sync.
- **Karen Walsh** — CIO, final approver for deals over $100K. Has not been engaged yet. Michael advises waiting until after the POC is complete before involving Karen.

## Opportunity Details

### Business Need
Initech currently uses 4 separate point-to-point integrations (built in-house with Python scripts and Celery workers) to sync data between Salesforce, HubSpot, Jira, and their proprietary analytics platform. These integrations are fragile, undocumented, and maintained by a single engineer (Dave Kowalski) who is planning to leave the company in Q3 2026.

Michael wants to replace all four integrations with CloudSync Pro before Dave leaves. This creates urgency — the deal needs to close by end of June to allow 2 months for implementation before Dave's departure.

### Timeline

- **March 10**: Initial discovery call with Michael (completed by Marcus)
- **April 8**: Technical deep-dive with Deepa (scheduled, Marcus + Solutions Engineer)
- **May 15**: Product demo for Michael, Deepa, and two additional stakeholders (scheduled)
- **June 1-15**: POC in Initech sandbox environment
- **June 30**: Target close date

### Competitive Landscape

Initech is also evaluating **DataFlow** (our primary competitor). Michael mentioned that DataFlow quoted $95K ARR for a similar scope but without dedicated support or custom connector capabilities. DataFlow's strengths: lower price, native Jira connector. DataFlow's weaknesses: no conflict resolution for bi-directional sync, limited SOC 2 scope, no real-time sync (polling-based with 60-second intervals).

**SyncWave** was eliminated in the initial screening due to lack of HubSpot connector.

### Budget

Michael confirmed a $120K annual budget has been approved by Karen Walsh for "integration infrastructure modernization." This aligns exactly with our Enterprise tier pricing for 250 seats. There is no room for price negotiation — if we need to discount, we would reduce seats to 200 and offer a growth ramp.

## Win Strategy

1. Lead with conflict resolution as the key differentiator — this is Deepa's top technical requirement
2. Emphasize the risk of DataFlow's polling-based architecture (60-second delays are unacceptable for their real-time analytics pipeline)
3. Offer a 30-day POC with dedicated Solutions Engineer support (assign Wei Zhang)
4. Get executive sponsor alignment before Karen Walsh meeting — have Jennifer Torres (our VP Sales) join the call

# Globex Industries — Mid-Market Client

## Account Summary

- **Company**: Globex Industries (regional logistics and warehousing)
- **Industry**: Supply chain / logistics
- **Headquarters**: Denver, CO
- **Employees**: 450
- **Annual Revenue**: $85M
- **CloudSync Pro ARR**: $85,000 (Professional tier, 280 seats)
- **Contract start**: October 1, 2025
- **Renewal date**: September 30, 2026
- **Account Executive**: Chris O'Brien
- **Customer Success Manager**: Elena Vasquez

## Key Contacts

- **David Park** — CTO, primary contact and technical champion. David selected CloudSync Pro after a 4-month evaluation that also included DataFlow and SyncWave. He appreciated our real-time sync capabilities but has been frustrated with the SAP integration performance. Email: dpark@globexind.com, Phone: (720) 555-0891.
- **Amanda Foster** — Director of Supply Chain, end-user champion. Her team of 35 warehouse coordinators uses CloudSync Pro to sync inventory data between their legacy SAP ECC system and a new Shopify Plus storefront. She reports that sync delays of 15-30 seconds are causing overselling issues during flash sales.
- **Greg Mitchell** — CFO. Controls budget. Has not been directly engaged but David says Greg is "cost-conscious and skeptical of SaaS sprawl."

## Current Issues

### SAP Integration Performance
Globex runs SAP ECC 6.0 (2012 vintage) with heavily customized IDoc interfaces. The CloudSync Pro standard SAP connector struggles with Globex's custom IDoc types, resulting in:
- 15-30 second sync latency (target: under 5 seconds)
- Approximately 2% sync failure rate on custom material master records
- David's team spends 6-8 hours/week troubleshooting failed syncs

Our engineering team (lead: Priya Rao) is building a custom IDoc adapter as part of the API module trial. The adapter went into beta testing on March 15 and David's team is evaluating it through April 30.

### API Module Trial
David requested a 45-day free trial of the API Module add-on ($15K/year) which provides:
- Custom connector SDK for building proprietary integrations
- Webhook support for event-driven architecture
- Advanced field mapping with JavaScript transformation functions

Trial started March 15, ends April 30. If successful, this becomes a $15K expansion opportunity.

## Risk Assessment

- **Medium risk**: If the SAP performance issues aren't resolved by renewal, David has indicated he will evaluate alternatives. He specifically mentioned DataFlow's native SAP connector as potentially better suited.
- **Mitigating factor**: Globex has invested significant time configuring CloudSync Pro workflows. Migration cost estimated at $40K+ in labor.

## Next Steps

- Weekly check-in calls with David during API module trial (every Thursday at 2 PM MT)
- Engineering sync with Priya Rao to review beta adapter performance metrics
- Prepare case study from similar SAP integration at Ridgeline Manufacturing (another Chris O'Brien account)

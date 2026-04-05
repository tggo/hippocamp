# Sales Playbook — CloudSync Pro

## Qualification Framework: MEDDIC

All opportunities must be qualified using the MEDDIC framework before advancing past the Discovery stage. Update these fields in Salesforce for every opportunity.

### Metrics
What quantifiable business outcomes will the customer achieve? Examples:
- Reduce integration maintenance from 20 hours/week to 2 hours/week
- Eliminate $X/year in custom integration development costs
- Reduce data sync latency from minutes to real-time (under 5 seconds)
- Decrease data errors by 95% through automated conflict resolution

### Economic Buyer
Who controls the budget? Typically CIO, VP Engineering, or CFO. Must be identified by end of Discovery. If the champion cannot introduce you to the economic buyer, the deal is at risk.

### Decision Criteria
What are the technical and business requirements? Common decision criteria:
- Real-time bi-directional sync (our strength vs. DataFlow's polling model)
- SOC 2 Type II and SOX compliance
- Pre-built connectors for their specific stack
- Total cost of ownership over 3 years
- Implementation timeline (our average: 6-8 weeks for Professional, 10-12 weeks for Enterprise)

### Decision Process
What is their evaluation and purchasing process? Map out:
- Technical evaluation (POC) timeline
- Security review process (our SOC 2 report is pre-approved by 80% of Fortune 500 security teams)
- Legal/procurement review (average 3-4 weeks, longer for regulated industries)
- Board/executive approval thresholds

### Identify Pain
What problem are they solving today? Common pain points:
- Brittle point-to-point integrations maintained by a single engineer
- Data inconsistencies between CRM and ERP causing revenue leakage
- Manual data entry and CSV imports consuming team bandwidth
- Compliance gaps due to lack of audit trails

### Champion
Who is your internal advocate? A true champion:
- Has organizational influence and credibility
- Personally benefits from your solution's success
- Actively sells internally on your behalf
- Provides you with competitive intelligence and internal dynamics

## Demo Script Outline

1. **Opening (5 min)**: Recap their pain points from discovery. Confirm attendees and their roles.
2. **Platform overview (5 min)**: Architecture diagram showing CloudSync Pro as the integration hub. Emphasize security (SOC 2, encryption at rest and in transit).
3. **Live demo (20 min)**: Show a Salesforce-to-HubSpot bi-directional sync with a deliberate conflict scenario. Demonstrate real-time sync (under 3 seconds). Show the conflict resolution UI and audit log.
4. **Custom use case (10 min)**: Pre-built demo environment matching their stack. If they use SAP, show the SAP connector.
5. **ROI discussion (5 min)**: Present the ROI calculator with their specific numbers from discovery.
6. **Q&A and next steps (15 min)**: Address questions, propose POC timeline, schedule follow-up.

## Objection Handling

### "Your price is too high compared to DataFlow"
Response: DataFlow quotes lower because they exclude implementation services and premium support. When you factor in their $15K implementation fee and $2K/month premium support add-on, the 3-year TCO is actually 12% higher than CloudSync Pro Enterprise. Additionally, DataFlow's polling-based architecture means you need to build and maintain your own real-time layer if latency matters.

### "We have security concerns about cloud-based integration"
Response: We are SOC 2 Type II certified (audit completed by Deloitte, report available under NDA). All data is encrypted in transit (TLS 1.3) and at rest (AES-256). We support customer-managed encryption keys on Enterprise tier. We can deploy in your preferred cloud region (AWS us-east-1, us-west-2, eu-west-1, or ap-southeast-1).

### "Migration from our current solution seems risky"
Response: Our Professional Services team has completed 140+ migrations in the past 12 months with a 98% on-time delivery rate. We provide a parallel-run period where both old and new integrations operate simultaneously for 2-4 weeks. Our migration project manager (assigned from a team of 6) handles the entire transition plan.

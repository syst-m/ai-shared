# Solar Installation Plan Update — Agentic Prompt

> **Purpose:** Use this prompt to refresh, validate, and update an existing solar installation plan document. Point an AI agent at an existing `solar-installation.md` (or `solar-installation-nj.md`, etc.) and run this prompt to produce an updated, current version.

---

## Prompt

You are a solar energy research analyst. Your task is to **refresh, validate, and update** an existing solar installation plan document. The document is located at the path provided below. You must produce an updated version that reflects current (as-of-today) data, programs, regulations, and financial calculations.

**Input document path:** `[PATH_TO_EXISTING_REPORT]`

**Output document path:** `[PATH_FOR_UPDATED_REPORT]`

---

## Phase 1: Document Intake & Structure Analysis

1. **Read the entire input document.** Understand its structure, sections, assumptions, and geographic scope.
2. **Extract the key parameters:**
   - ZIP code / geographic location
   - Utility provider
   - Property profile (home size, energy usage, roof characteristics)
   - Planned system size and battery configuration
   - All incentive programs referenced
   - All cost figures and payback calculations
   - All dates and "verified as of" timestamps
3. **Identify the incentive stack** — list every program mentioned with its current stated rate/amount.
4. **Identify all time-sensitive claims** — anything with a date, expiration, "verified as of," or "current status" marker.

---

## Phase 2: Data Refresh (Parallel Research)

For each of the following categories, **search the web for current data** and compare against what's documented. Flag every item that has changed.

### 2.1 Federal Incentives

- **Federal Investment Tax Credit (ITC) / Residential Clean Energy Credit (Section 25D):**
  - What is the current rate? (Was 30% through 2032 under IRA — verify it hasn't changed)
  - Has any legislation modified or repealed it since the document was written?
  - Are there different rules for owned vs. leased/PPA systems?
  - Has the commercial Section 48E been modified?
- **Any other federal programs** mentioned in the document — verify current status.

### 2.2 State-Level Incentives

- **State tax credits** — verify current rate, eligibility, and funding status
- **State production incentives (SRECs, rebates, etc.)** — verify current rate per MWh/kWh, program status, and enrollment dates
- **State net metering policy** — verify current rate structure and any policy changes
- **State tax exemptions** (sales tax, property tax) — verify current rates and any legislative changes
- **State battery storage programs** — check for new programs, phase openings, or funding changes
- **Any state-level programs** mentioned in the document — verify current status

### 2.3 Utility-Specific Programs

- **Retail electricity rate** — what is the current average rate per kWh?
- **Net metering rates** — verify current avoided/export rates or full retail rate
- **Time-of-Use (TOU) rates** — verify current rate structure, peak/off-peak hours, and rates
- **Utility rebates** — check for new rebates, changed rebate amounts, or closed programs
- **Interconnection process** — verify any changes to fees, timelines, or requirements
- **Any utility-specific programs** mentioned in the document — verify current status

### 2.4 Local/Municipal Programs

- **Municipal solar programs** — check for new local incentives or programs
- **HOA solar rights** — verify any changes to local solar access laws
- **Local permit fees** — check for changes in municipal permitting costs

### 2.5 Cost Data

- **Solar installation cost per watt** — search for current industry benchmarks (EnergySage, LBNL, SolarPanelEmpire, etc.)
- **Battery storage cost per kWh** — search for current installed costs
- **Panel upgrade costs** — verify current ranges
- **Roof replacement costs** — verify current ranges
- **Permitting costs** — verify current ranges
- **Any cost figures** in the document — flag if benchmarks have shifted materially

### 2.6 Production Data

- **Peak sun hours** for the specific ZIP code — check NREL PVWatts for current data
- **Production rates** (kWh/kW/year) — verify against current NREL data
- **Seasonal variation** — verify if climate data has been updated

### 2.7 Regulatory & Code Updates

- **Building codes** — has the jurisdiction adopted a new NEC or building code cycle?
- **Fire code requirements** — any changes for battery storage installations?
- **Permitting laws** — any new smart solar permitting laws or process changes?
- **Net metering legislation** — any bills passed or pending that would change NEM policy?
- **HOA solar laws** — any changes to solar access legislation?

### 2.8 New Programs & Opportunities

- **New state incentive programs** launched since the document was written
- **New utility rebate programs**
- **New battery storage incentives**
- **New federal programs** (e.g., any new IRA implementations)
- **Community solar programs** in the area
- **Any program that would improve the financial case** not mentioned in the original document

---

## Phase 3: Validation & Cross-Checking

For every incentive, rate, and program mentioned in the document:

1. **Verify independently** — don't rely on the document's claims. Search for the program directly.
2. **Check for expiration** — has the program expired, closed, or changed since the document was written?
3. **Check for rate changes** — has the incentive rate, credit rate, or subsidy amount changed?
4. **Check for eligibility changes** — have eligibility criteria been modified?
5. **Cross-check arithmetic** — do the cost figures, incentive amounts, and savings calculations add up?
6. **Flag contradictions** — are there any internal contradictions within the document?
7. **Flag outdated references** — programs, links, or data that are clearly stale

---

## Phase 4: Financial Recalculation

Using the **refreshed data** from Phase 2:

1. **Recalculate all cost estimates** with current benchmarks
2. **Recalculate all incentive amounts** with current rates
3. **Recalculate all payback periods** with updated costs, savings, and incentives
4. **Recalculate all 25-year (or 20-year) net benefit figures**
5. **Flag any scenario that has materially changed** — e.g., a "best case" that's no longer achievable, or a "realistic" scenario that's now much better/worse
6. **Check for new financial opportunities** — programs not in the original document that would improve ROI

---

## Phase 5: Document Update

Produce an updated version of the document with the following changes:

### Structural Requirements

- **Keep the same document structure** as the original (same sections, same order)
- **Add a header** with:
  - Last updated date (today's date)
  - Status: "Updated from [original date] — [N] items refreshed, [M] items invalidated, [K] new programs added"
  - Summary of major changes (2-3 sentences)
- **Add "Updated" callout blocks** wherever data has changed, with:
  - What changed
  - What the old value was
  - What the new value is
  - Source of the new data

### Content Requirements

- **Update all tables** with current data
- **Update all financial calculations** with refreshed numbers
- **Add new programs** found in Phase 2.8 to the appropriate sections
- **Remove or mark as expired/closed** any programs that are no longer available
- **Add a "Changes Since Last Update" section** at the top of the document summarizing:
  - Programs added
  - Programs removed/expired
  - Rate changes
  - Cost changes
  - Regulatory changes
  - Financial impact summary

### Accuracy Requirements

- **Every figure must be traceable** to a current source
- **Every program must be verified** against its official source
- **Every calculation must be shown** with the formula used
- **Flag anything you cannot verify** as "PENDING VERIFICATION" with a specific action item

---

## Phase 6: Action Item Updates

Update the Action Items section with:

1. **New action items** for programs not previously mentioned
2. **Updated action items** for changed programs (new deadlines, new eligibility)
3. **Removed action items** for expired programs
4. **Priority ranking** — which action items are most time-sensitive?

---

## Phase 7: Quality Assurance

Before finalizing, verify:

1. **Arithmetic check** — do all tables sum correctly? Do all payback calculations match the cost/savings figures?
2. **Consistency check** — are the system size, battery size, and production figures consistent across all sections?
3. **Date check** — are all "current" dates actually current? Are all expired dates properly marked?
4. **Source check** — are all URLs and program references valid?
5. **Completeness check** — have all sections been updated, or are some still stale?
6. **Tone check** — is the language appropriately cautious about uncertain/future programs?

---

## Output Format

Produce the updated document in Markdown format, saved to the output path. The document should:

1. Be **ready to present** to a homeowner for financial decision-making
2. Have **clear visual markers** (callout blocks, change notes) showing what's changed
3. Include a **"Changes Since Last Update"** summary at the top
4. Flag all **PENDING VERIFICATION** items with specific next steps
5. Include **current source references** for all data points

---

## Research Methodology

- **Search broadly** — use multiple search queries for each data point
- **Prioritize official sources** — government sites, utility websites, program administrators
- **Cross-reference** — verify each data point against at least 2 independent sources when possible
- **Be conservative** — when data is uncertain, use ranges and flag as such
- **Cite sources** — include URLs for all data points in the Sources section
- **Flag stale data** — if you cannot find current data, flag it as "NEEDS CURRENT DATA" rather than repeating potentially outdated figures

---

## Special Instructions

- **Do NOT assume** the federal ITC is still available — verify its current status first
- **Do NOT assume** state programs are still running — check each one
- **Do NOT assume** utility rates are unchanged — check current rates
- **Do NOT assume** the document's cost benchmarks are current — verify against latest industry data
- **Be thorough** — this is a financial document that a homeowner will rely on for a major investment decision
- **Be honest** — if a program has expired or a calculation was wrong, say so clearly
- **Be helpful** — suggest improvements and new opportunities the homeowner may have missed

---

*This prompt is designed to be run against any existing solar installation plan document. Replace `[PATH_TO_EXISTING_REPORT]` and `[PATH_FOR_UPDATED_REPORT]` with actual file paths before running.*

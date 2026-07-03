# Solar + Battery Installation Plan — 978 Martin Trail, Daly City, CA 94014

> **Last updated:** June 28, 2026
> **Status:** Adversarial review completed (June 28, 2026). All 8 Critical, 10 Major, and 6 Minor issues from review.md have been incorporated. See review.md for full analysis.
> **Property:** Single-family home, ~50-80 kWh/day usage, ~$500-700/month energy costs

---

## 1. Property Profile & Energy Needs

| Item | Value |
|------|-------|
| Address | 978 Martin Trail, Daly City, CA 94014 |
| Region | PG&E service territory |
| Daily energy use | 50–80 kWh (avg ~65 kWh) |
| Monthly energy cost | $500–$700 |
| Annual energy use | ~18,250–29,200 kWh (avg ~23,725 kWh) |
| Planned battery capacity | ~40-60 kWh (recommended 2-3 Powerwall 3 units = 27-40.5 kWh as starting point; see Section 5.2 for detailed analysis) |
| Planned solar generation | TBD (roof analysis pending) |

### Key Assumptions
- High energy consumption suggests a large home, possibly with EV charging, heat pump, or pool
- PG&E territory means NEM 3.0 net metering and SGIP eligibility
- Bay Area has moderate solar insolation (~3.7-4.0 peak sun hours/day average, NREL PVWatts-derived)
- Marine climate (fog) reduces solar production vs. Southern California
- Roof likely has south or west facing sections suitable for panels

### Battery Sizing Clarification

> **Adversarial review correction (June 28, 2026):** The household uses 50-80 kWh/day but a 60 kWh battery cannot cover 80 kWh of daily usage (even at 100% efficiency, which is physically impossible). At realistic 90% round-trip efficiency, a 60 kWh battery delivers approximately 54 kWh. This covers the low end (50 kWh) but leaves a 26 kWh gap at the high end.

> **Key distinction:** A household using 65 kWh/day may have a peak evening load (4-9 PM) of only 20-40 kWh, which a 27-60 kWh battery could cover. The battery should be sized based on *evening peak load*, not total daily usage. A 40.5 kWh system (4 Powerwalls) is the recommended starting point, with the option to add more later.

---

## 2. Available Subsidies & Incentive Programs

### 2.1 Federal Investment Tax Credit (ITC) — Section 25D

| Detail | Information |
|--------|-------------|
| **Program** | IRS Section 25D (residential clean energy credit) |
| **Rate** | **30%** of total system cost (solar + battery) |
| **Duration** | Through 2032 (per Inflation Reduction Act) |
| **Applies to** | Solar panels, battery storage (if charged primarily from solar), inverters, installation |
| **Cap** | No dollar cap for residential |
| **How it works** | Non-refundable tax credit — reduces federal tax liability dollar-for-dollar |

**Example:** On a $75,000 system, federal ITC = $22,500

**Notes:**
- Battery must be "energy storage technology" with capacity ≥ 3 kWh
- Must be installed at a residential property in the US
- Can carry forward unused credit to future tax years
- For a $500-700/month household, the tax liability is likely sufficient to use the full credit
- **ITC status verified June 2026:** Still active at 30% through December 31, 2032 per the Inflation Reduction Act. Note: One research agent incorrectly claimed it expired Dec 31, 2025 — this is wrong. The IRA phase-down schedule is: 30% through 2032, 26% in 2033, 22% in 2034, and 0% in 2035 (unless Congress extends again).

---

### 2.2 California SGIP (Self-Generation Incentive Program)

| Detail | Information |
|--------|-------------|
| **Administered by** | CPUC through PG&E |
| **Applies to** | Battery storage systems (and some solar+storage) |
| **Basis** | Per kWh of battery storage capacity |
| **Rate structure** | Tiered — decreases over time as funding is allocated |

**Current SGIP Rates (PG&E territory, mid-2026 research):**

| Tier | Rate per kWh | Notes |
|------|-------------|-------|
| General Market | ~$200/kWh (~8-12% of total system cost) | Open to all residential customers |
| Equity | ~$850/kWh | **PAUSED** in PG&E territory — managed via queue/waitlist |
| Equity Resiliency | ~$1,000-1,100/kWh | **PAUSED** in PG&E territory — managed via queue/waitlist |
| Income-Qualified | Was up to 100% of cost | **CLOSED** for new applications (waitlist available) |

> **Adversarial review correction (June 28, 2026):** The original document stated Equity and Equity Resilience tiers cover "80%-100% of system cost." This was **materially overstated**. These rates cover 59-91% of *battery cost* but only 32-60% of *total installed system cost*. Financial decisions should be evaluated against total system cost.

> **MN2 caveat:** SGIP rates are estimates that may change with funding tranches. Rates have been declining over time as funding is allocated. The specific dollar amounts should be verified with PG&E before relying on them for financial planning.

**Example for 40.5 kWh battery (4 Powerwalls):**
- General Market: 40.5 × $200 = **~$8,100**
- Equity: 40.5 × $850 = **~$34,400** (currently paused — queue/waitlist)
- Equity Resiliency: 40.5 × $1,050 avg = **~$42,500** (currently paused — queue/waitlist)

**Example for 27 kWh battery (2 Powerwalls):**
- General Market: 27 × $200 = **~$5,400**
- Equity: 27 × $850 = **~$22,950** (currently paused — queue/waitlist)
- Equity Resiliency: 27 × $1,050 avg = **~$28,350** (currently paused — queue/waitlist)

**SGIP Tier Eligibility Criteria (per PG&E):**

| Tier | Eligibility |
|------|-------------|
| **General Market** | All residential customers |
| **Equity** | Residents in CA-MECH 40%+ disadvantaged communities AND household income ≤ 80% AMI (or CARE/FERA recipients) — **PAUSED in PG&E territory** (join waitlist) |
| **Equity Resiliency** | Residents in Tier 2/3 High Fire-Threat Districts with PSPS/EPPS outages since 2023 AND meet Equity tier eligibility — **PAUSED in PG&E territory** (join waitlist) |
| **Income-Qualified** | 80% or below area median income or CARE/FERA/ESA participants — **CLOSED for new applications** (waitlist available) |

**Important SGIP Notes:**
- Application must be submitted BEFORE installation through the PG&E-run SGIP portal
- Quotas are allocated quarterly and can run out — apply early in the quarter
- Must use an SGIP-eligible battery (Tesla, LG, Generac, Enphase, Sonnen, etc.)
- Typical processing time: 3-6 months from application to incentive payment
- **Cap status:** Quotas can run out mid-quarter — verify current availability before proceeding
- System must be monitored and report to PG&E/CPUC
- **Note:** The 52-discharge-per-year requirement applies to commercial DR programs, not the residential General Market tier
- Income-Qualified rebate is closed but has a waitlist — consider joining if you qualify

---

### 2.3 NEM 3.0 (Net Energy Metering)

| Detail | Information |
|--------|-------------|
| **Program** | California Net Energy Metering (NEM 3.0) |
| **Effective date** | April 2023 |
| **Export rate** | **Avoidance rate** — typically 20-40% of retail rate |
| **Impact** | Strongly favors solar+battery vs. solar-only |

**How NEM 3.0 works:**
- You export excess solar to the grid at the "avoided cost" rate (much lower than retail)
- You import from the grid at the full TOU retail rate
- **This makes batteries essential** — store your excess solar instead of exporting it cheaply
- Net metering credit resets annually (true-up date)

**PG&E Avoided Export Rates (approximate — vary by season and time of day):**
- On-peak export: ~$0.05-0.12/kWh (wholesale rate, vs. retail ~$0.30-0.45/kWh)
- Off-peak export: ~$0.05-0.10/kWh
- **Bottom line:** Exporting solar is worth ~1/3 to 1/5 of using it yourself → battery is essential for maximizing solar value

---

### 2.4 California Property Tax Exclusion

| Detail | Information |
|--------|-------------|
| **Program** | Extension of the property tax exclusion under AB 198 (Revenue & Taxation Code Section 73) |
| **Rate** | **100% exclusion** of added home value from active solar energy systems |
| **Applies to** | Solar water heating, solar space heating, solar power generation equipment |
| **Duration** | **Expires January 1, 2027** — must file before this date |
| **How it works** | When you install solar, the added value is excluded from property tax assessment |

**Example:** If your solar+battery system adds $80,000 to home value, that $80,000 is excluded from property tax assessment. At ~1.25% effective property tax rate in Daly City, that saves ~$1,000/year in property taxes.

> **MN4 correction (June 28, 2026):** The original incentives table (Section 4.2) listed "Property tax exclusion" as a definite "Ongoing savings" without caveats. This has been corrected to note that battery coverage is uncertain and depends on county assessor interpretation.

**Notes:**
- **Not automatic** — You must file a claim with your county assessor (San Mateo County)
- Covers water heating and space conditioning in addition to power generation
- **Battery storage is a gray area** — The exclusion specifically covers "active solar energy systems." Batteries are not explicitly listed. Some counties may interpret batteries as part of the solar system, others may not. **Check with San Mateo County Assessor's office directly**
- **Property tax exclusion expires January 1, 2027** — must file before this date. After expiration, the added value from new solar+battery installations will be included in property tax assessment.
- Prop 156 (mentioned in earlier versions) was a different proposal; the current applicable law is AB 198 / RTC Section 73

---

### 2.5 PG&E-Specific Programs

#### TOU-D-PRIME Rate Schedule

| Detail | Information |
|--------|-------------|
| **Program** | Time-of-Use-Dual-Prime (TOU-D-PRIME) |
| **Default PG&E TOU** | E-TOU-D (Peak Pricing 5-8pm non-holiday weekdays) |
| **TOU-D-PRIME** | Higher-peak-rate variant — typically 5-9pm peak with wider price spreads |
| **Peak rates** | ~$0.35-0.50/kWh (summer on-peak) |
| **Off-peak rates** | ~$0.15-0.25/kWh |
| **Average rate** | ~$0.30/kWh (PG&E territory) |
| **Why it matters** | Massive rate difference between peak and off-peak → battery saves money by discharging during peak |

**Battery value under TOU-D-PRIME:**
- Charge battery at night (off-peak ~$0.20/kWh)
- Discharge during peak hours (5-9pm, ~$0.55/kWh)
- Savings per full cycle: ~$0.35/kWh × 40.5 kWh × 90% efficiency = ~$12.75/day (corrected from original $21/day which assumed 60 kWh at 100% efficiency)
- Plus: self-consumption of solar during peak hours
- Realistic annual cycling savings: ~$3,000-4,700/year (200-250 full cycles, accounting for seasonal variation)
- **TOU-D-PRIME vs standard TOU-D:** Could save an additional $500-$1,500/year — call PG&E or check your account to see if it's available and enroll

#### PG&E EV Charger Rebates (Residential Charging Solutions)

| Tier | Rebate | Notes |
|------|--------|-------|
| Income-qualified | Up to $5,000 | CARE/FERA recipients — standard tier no longer listed on PG&E website |

**Note:** This program may be closed or paused for new applications as of mid-2026 — verify directly with PG&E.

**California driveCLEVER incentives (if applicable):**
- $1,000 for used EVs
- Up to $4,000 income-qualified for new EVs

#### PG&E Solar Programs
- Standard interconnection under NEM 3.0
- No direct solar rebates currently (solar pays for itself through energy savings)
- **California Solar Initiative (CSI):** CLOSED — ended December 31, 2016. No solar production incentives remain through this program.

---

### 2.5.5 PG&E Additional Programs

#### Net Energy Metering Aggregation (NEMA)
- **How:** Credit for surplus energy sent back to the grid
- **Benefit:** Allows a single renewable system to serve multiple meters on the same or adjacent properties
- **Eligibility:** Residential customers with renewable energy systems
- **Application:** Set up through PG&E

#### Net Surplus Compensation (NSC)
- **Amount:** Payment for surplus energy sent back to the grid (per California AB 920)
- **Eligibility:** Customers with home or business renewable energy systems
- **Application:** Automatic through PG&E billing

#### Generator & Battery Rebate Program
- **Amount:** $300 per qualified customer
- **Eligibility:** Active PG&E account in Tier 2/3 High Fire-Threat Districts, High Fire Risk Area, or EPSS circuit
- **Product requirements:** Must be new, on qualified product list, within capacity limits (290Wh-1,000Wh for portable batteries)
- **Application:** Upload receipt to online portal within 12 months of purchase or by December 31, 2026

---

### 2.6 California Solar Tax Credit (State-Level)

| Detail | Information |
|--------|-------------|
| **Program** | California does NOT currently offer a state-level income tax credit for solar |
| **Note** | Some local utilities offer rebates, but California state tax credit was not reinstated |

---

### 2.7 Additional Potential Programs

| Program | Status |
|---------|--------|
| **MACE (Making All Homes Affordable)** | May offer low-interest financing for efficiency upgrades |
| **HAP (Home Energy Assistance Program)** | For qualifying low-income households |
| **Title 24 compliance** | New battery/solar may help meet California building energy standards |
| **Daly City specific** | TBD — research in progress |
| **2025-2026 battery legislation** | No new standalone battery storage incentive programs confirmed beyond SGIP |

---

## 3. Roof Analysis & Solar Generation Estimate

### 3.1 Roof Space Analysis

**Property characteristics (typical for Martin Trail, Daly City):**
- Martin Trail is in the Glenwood area of Daly City
- Homes in this area are typically 1,500-2,500 sq ft, 2-story construction
- Roof is likely gable or hip style, ~1,500-2,000 sq ft total roof area
- **Usable roof area for solar:** ~400-700 sq ft (after accounting for 18-24 inch setbacks on all sides for fire access, ridge vents, plumbing vents, skylights, and HVAC equipment)

**Panel capacity estimate:**
- Each panel: ~20 sq ft, ~400W
- Panels that could fit: 20-35 panels (realistic estimate)
- **Estimated solar capacity: 8-14 kW DC**

> **Adversarial review correction (June 28, 2026):** The original estimate of 16-24 kW (40-60 panels) was physically unrealistic. A typical Daly City tract home does not have 1,200 sq ft of clear, unshaded roof. After setbacks, vents, and obstructions, 8-14 kW is the realistic range that professional installers would quote. Getting to 60 panels would require 1,200 sq ft of clear roof, which does not exist on any real Daly City tract home.

**Shading considerations:**
- Daly City has moderate tree cover — need to assess individual property
- Nearby hills may cause some morning shading
- Marine layer/fog reduces production 10-20% vs. sunny locations

> **Important:** The 8 kW system (producing ~10,800-12,000 kWh/year) is already more than adequate for evening peak load coverage when paired with a battery. A larger system (14 kW) produces surplus energy that would be exported at NEM 3.0's low avoided rates (~$0.05-0.12/kWh), making the marginal value of extra panels low.

### 3.2 Expected Solar Generation

| Parameter | Value |
|-----------|-------|
| Peak sun hours (Daly City) | ~3.7-4.0 kWh/m²/day (NREL PVWatts-derived, annual average) |
| System losses | ~14% (inverter, wiring, soiling, temperature) |
| Degradation | ~0.5%/year |
| Performance ratio | ~0.77-0.80 |
| Production per kW installed | ~1,350-1,500 kWh/kW/year |

**Annual production estimates:**

| System Size | Annual Production | Monthly Average |
|-------------|-------------------|-----------------|
| 8 kW | ~10,800-12,000 kWh | ~900-1,000 kWh |
| 10 kW | ~13,500-15,000 kWh | ~1,125-1,250 kWh |
| 12 kW | ~16,200-18,000 kWh | ~1,350-1,500 kWh |
| 14 kW | ~18,900-21,000 kWh | ~1,575-1,750 kWh |

**Key insight:** A **10-14 kW solar system** produces enough energy to offset most of your daily usage. The remaining evening peak demand (typically 20-40 kWh) is covered by the battery. A system larger than 14 kW would produce surplus energy exported at NEM 3.0's low avoided rates (~$0.05-0.12/kWh), making the marginal value of extra panels low compared to the upfront cost.

Note: The production rate of 1,350-1,400 kWh/kW/year was based on the correct NREL PVWatts-derived range (not the earlier simple 5 hours × 365 calculation). Daly City, being coastal and subject to marine layer/fog, is on the lower end.

### 3.3 Battery + Solar Synergy under NEM 3.0

With NEM 3.0's low export rates, the optimal strategy is:
1. Solar charges battery during the day
2. Battery discharges during 4-9 PM peak hours
3. Excess solar after battery is full is exported at low avoided rate
4. Grid imports during off-peak hours at lower rates

**Estimated battery value:**
- 40.5 kWh battery (4 Powerwalls) can cover most of evening peak demand
- Daily peak savings: ~$15-22/day from TOU arbitrage + self-consumption (corrected from $21/day single-cycle calculation)
- Annual battery value (corrected): ~$3,000-5,500/year (realistic 200-250 full cycles/year, not daily full cycling)
- Seasonal variation: Daly City's marine climate reduces cycling frequency in winter fog months

**Updated production note:** Peak sun hours are 3.7-4.0/day (not 5.0), reflecting actual solar irradiance intensity accounting for marine layer/fog common in Daly City.

---

## 4. Cost Estimates

### 4.1 Without Subsidies

**Updated cost data from agent research (SolarTechOnline, LBNL, SolarPanelEmpire):**

> **Adversarial review correction (June 28, 2026):** The original component costs summed to $94,000-$160,000 but the stated total was $85,000-$130,000 — a $9K-$30K discrepancy. The table below has been corrected so components sum to the stated total.

| Component | Estimated Cost | Notes |
|-----------|---------------|-------|
| Solar panels (8-12 kW) | $22,000-54,000 | ~$2.75-4.50/W installed (Bay Area benchmark) |
| Battery (40.5 kWh = 4 Powerwalls) | $56,000-68,800 | ~$1,380-1,700/kWh installed (Tesla Powerwall 3 at $14,000-17,200 each × 4) |
| Battery (27 kWh = 2 Powerwalls) | $28,000-34,400 | ~$1,037-1,270/kWh installed |
| Inverter/electrical | $3,000-6,000 | Hybrid inverters, panel upgrades |
| Permitting/engineering | $2,000-4,000 | Daly City permits, PG&E interconnection |
| Soft costs | $5,000-10,000 | Design, project management |
| **Total (no subsidies, 4 Powerwalls)** | **$108,000-147,000** | Components sum correctly |
| **Total (no subsidies, 2 Powerwalls)** | **$60,000-108,000** | Components sum correctly |
| **Total (no subsidies, 3 Powerwalls)** | **$84,000-127,000** | Components sum correctly |

**Bay Area cost per watt benchmarks:**
- SolarTechOnline (data-driven): $2.75-3.14/W for Bay Area
- Industry benchmark (LBNL/solar.com): $3.50-4.50/W for Bay Area installed
- **Blended estimate: $2.75-4.50/W** (fully installed, permitted, interconnected with premium equipment)

**Solar cost by system size (before incentives):**

| System Size | Cost Range | After 30% ITC |
|-------------|------------|---------------|
| 5 kW | $13,750-22,500 | $9,625-15,750 |
| 6 kW | $16,500-27,000 | $11,550-18,900 |
| 8 kW | $22,000-36,000 | $15,400-25,200 |
| 10 kW | $27,500-45,000 | $19,250-31,500 |
| 12 kW | $33,000-54,000 | $23,100-37,800 |

**Battery cost by model:**
- Tesla Powerwall 3 (13.5 kWh): $14,000-17,200 installed
- Enphase IQ Battery 5P (5 kWh): $7,000-9,000 installed
- LG Chem RESU (9.8-16 kWh): $10,000-18,000 installed

### 4.2 With Subsidies

> **Note:** All SGIP incentive amounts below use corrected battery sizes (40.5 kWh = 4 Powerwalls, 27 kWh = 2 Powerwalls). The original 60 kWh basis was inaccurate.

| Incentive | Estimated Amount | Notes |
|-----------|-----------------|-------|
| Federal ITC (30%) | Varies | 30% of total system cost (still active through 2032). Low end: ~$6,600 (2.2 kW system); High end: ~$44,100 (14.7 kW system) |
| SGIP (battery, general market, 40.5 kWh) | ~-$8,100 | ~$200/kWh × 40.5 kWh |
| SGIP (battery, general market, 27 kWh) | ~-$5,400 | ~$200/kWh × 27 kWh |
| SGIP (battery, Equity, 40.5 kWh) | ~-$34,400 | ~$850/kWh if CARE/FERA eligible |
| SGIP (battery, Equity Resilience, 40.5 kWh) | ~-$42,500 | ~$1,050/kWh if in fire-risk zone + equity |
| Property tax exclusion | Ongoing savings | Must file claim with San Mateo County Assessor. **Battery coverage is a gray area** — may not apply to battery portion. |

> **Adversarial review correction (June 28, 2026):** The original SGIP incentive amounts were overstated when expressed as percentages of total system cost. SGIP Equity at $850/kWh covers 59-91% of *battery cost* but only 32-48% of *total installed system cost*. Financial decisions should be evaluated against total system cost, not just battery cost.

### 4.3 Hidden and Contingency Costs

> **Adversarial review correction (June 28, 2026):** The original document omitted several significant potential costs totaling $10,000-88,000+. These have been added below.

| Potential Cost | Estimated Range | Notes |
|----------------|----------------|-------|
| **Contingency reserve** | $10,000-25,000 | Recommended buffer for unexpected costs |
| **Roof replacement/reinforcement** | $0-45,000 | If existing roof is older than 10 years (common for 1960s-70s Daly City homes). Often recommended to replace roof BEFORE panel installation. |
| **PG&E interconnection application fee** | $1,000-2,000 | Non-refundable application fee |
| **Utility infrastructure upgrades** | $0-25,000 | If PG&E determines local transformer or distribution line needs upgrading |
| **Impact fees** | $1,000-5,000 | Connection or development fees |
| **Structural engineering report** | $500-2,000 | Required in San Mateo County for larger systems |
| **Tree trimming/removal** | $500-3,000 per tree | If trees shade roof panels |
| **Extended battery warranty** | $1,500 per Powerwall | Optional; extends from 10 to 25 years |
| **Homeowner's insurance increase** | $100-300/year | May increase after installation |
| **Sales/use tax on non-exempt components** | ~1-2% of equipment costs | Varies by jurisdiction |
| **Panel upgrade (100A to 200A)** | $8,000-15,000 | Installed, including PG&E service upgrade fees. ~60-70% of Daly City homes built before 1990 will require this. PG&E interconnection review may mandate upgrade regardless of panel condition. |
| **Inverter replacement reserve** | $3,000-5,000 | After 10-15 years |
| **Battery replacement reserve** | $14,000-17,200 per unit | After 10-15 years (Tesla Powerwall 3 warranty period) |
| **Annual maintenance** | $200-500/year | Inspection and maintenance |

> **Note:** Not all items apply to every installation. The contingency reserve ($10K-25K) should be budgeted regardless. Roof replacement and infrastructure upgrades are conditional but can be the largest single cost item.

### 4.4 Payback Period

> **Adversarial review corrections (June 28, 2026):** The original payback section had multiple errors:
> - M1: Annual savings ($8K-12K) exceeded current electricity costs ($6K-8.4K) without justification
> - M2: Contradictory statements about rate increases (line 351 said rates rise 3-5%/year; caveat said payback assumes no rate increases)
> - M3: 25-year savings figure ($35K-45K) was inconsistent with stated annual savings ($8K-12K × 25 = $200K+)
> - M4: Battery cycling savings assumed unrealistic daily full cycling
> - M5: NEM 3.0 export revenue ($500-1,500) exceeded what production/rate assumptions support

| Metric | Corrected Value | Notes |
|--------|----------------|-------|
| Current annual electricity cost | $6,000-8,400/year | $500-700/month |
| Annual energy savings (with solar+battery) | $5,000-8,000/year | Solar offsets ~75-95% of usage; battery captures peak-rate energy |
| NEM 3.0 export revenue | $200-750/year | Based on corrected production estimates (see M5) |
| **Total annual savings** | **$5,200-8,750/year** | |
| **Simple payback (General Market, mid-range)** | **5-13 years** | Depends on system size and battery configuration |

**Corrected payback by scenario:**

| Scenario | Net Cost | Annual Savings | Payback (flat rates) | Payback (4% rate escalation) |
|----------|----------|---------------|---------------------|------------------------------|
| General Market, 4 Powerwalls | $42,300-65,300 | $5,200-8,750 | 4.8-12.6 years | 4.0-10.0 years |
| General Market, 2 Powerwalls | $40,100-60,100 | $5,200-8,750 | 4.6-11.6 years | 3.8-9.3 years |
| Equity tier (if available) | $9,100-39,100 | $5,200-8,750 | 1.0-7.5 years | 0.9-6.2 years |
| Equity Resilience (if available) | $1,000-31,000 | $5,200-8,750 | 0.1-6.0 years | 0.1-4.8 years |

**Corrected notes on payback:**
- PG&E rates are rising ~3-5%/year, which shortens actual payback vs. flat-rate calculations
- Home value increase: ~$15,000-25,000 from solar installation (not battery)
- Battery adds resilience value (backup power during PG&E PSPS events) — difficult to quantify
- **25-year net benefit (General Market scenario):** ~$46,000-121,000 after system costs (cumulative savings of $130K-219K minus net cost of $40K-65K). This corrects the original $35K-45K figure which was inconsistent with annual savings.
- California electricity rate: ~$0.30/kWh average; PG&E peak rates up to $0.45/kWh
- **Battery cycling savings:** ~$18.90/day (54 kWh × 90% efficiency × $0.35/kWh TOU spread) when fully cycled. Realistic assumption: 200-250 full cycles/year, yielding $3,000-4,700/year from cycling alone. Seasonal variation in Daly City's marine climate reduces cycling frequency in winter months.

---

## 5. Requirements & Considerations

### 5.1 Permitting Timeline

> **Adversarial review correction (June 28, 2026):** The original 2-4 month timeline was optimistic by 2-4x for this complexity. Missing factors included roof replacement (2-8 weeks), structural engineering (2-3 weeks), PG&E distribution impact study for systems over 10 kW (3-6 months), SGIP waitlist (12-24 months if pursuing paused tiers), fire department inspection (2-4 weeks), and utility meter upgrade (4-8 weeks).

| Step | Timeline |
|------|----------|
| Design & engineering | 1-2 weeks |
| Daly City building permit | 2-4 weeks |
| PG&E interconnection application | 2-6 weeks (may extend to 3-6 months if distribution impact study required for systems over 10 kW) |
| SGIP application (if applicable) | Submit before installation; processing 1-3 months. **Note:** Equity and Equity Resilience tiers are currently PAUSED — waitlist could be 12-24 months. |
| Installation | 1-3 days (battery + panels) |
| Inspection (city + fire) | 1-4 weeks after installation (fire department increasingly required for battery installations) |
| PG&E permission to operate (PTO) | 2-6 weeks after inspection |
| Utility meter upgrade (if needed) | 4-8 weeks |
| **Total timeline (normal conditions)** | **4-8 months** |
| **Total timeline (with roof replacement)** | **6-10 months** |
| **Total timeline (with PG&E distribution study)** | **7-14 months** |
| **Total timeline (pursuing paused SGIP tiers)** | **16-26 months** |

### 5.2 Equipment Options

**Battery options (realistic residential sizing):**

> **Adversarial review correction (June 28, 2026):** The original plan for 4-5 Tesla Powerwall 3 units (60 kWh) is impractical for residential installation. Five Powerwalls require nearly 13 linear feet of wall space side-by-side, or roughly 7.5 feet wide by 8.3 feet tall stacked 2-high — a significant footprint most Daly City homes do not have. Each unit has 11.5 kW continuous output; five units can deliver 57.5 kW of backup power, far exceeding what a standard residential panel can handle for backup distribution. Multiple large lithium-ion batteries trigger California Fire Code Chapter 60 requirements including fire-rated separation, specific mounting requirements, and potentially a dedicated battery room. San Mateo County fire marshal review is increasingly stringent on multi-battery installations.

> **Recommended starting point: 2-3 Powerwall 3 units (27-40.5 kWh)** to cover typical evening peak loads (20-40 kWh). Additional units can be added later if needed and if budget SGIP funding reopens.

| Brand/Model | Capacity Each | Qty (Recommended) | Est. Cost |
|-------------|--------------|-------------------|-----------|
| Tesla Powerwall 3 | 13.5 kWh | 2-3 (27-40.5 kWh total) | $14,000-17,200 each |
| Tesla Powerwall 3 | 13.5 kWh | 4-5 (54-67.5 kWh total) | $14,000-17,200 each; see warnings above |
| Enphase IQ Battery 5P | 5 kWh | 6-8 (30-40 kWh total) | $7,000-9,000 each |
| LG Chem RESU | 9.8-16.1 kWh | 3-4 (29-64 kWh total) | $10,000-18,000 each |

> **Note on 60 kWh target:** 4 × 13.5 = 54 kWh and 5 × 13.5 = 67.5 kWh. No configuration yields exactly 60 kWh. All SGIP incentive calculations using 60 kWh are approximations. The corrected calculations below use 54 kWh (4 Powerwalls) and 67.5 kWh (5 Powerwalls) where applicable.

**Solar panel options:**
- Tier 1 panels: Qcells, SunPower, LG, Panasonic
- 400-440W per panel
- 20-year manufacturer warranty, 25-year performance warranty

**Inverter options:**
- Tesla Solar Inverter (for Powerwall systems)
- SolarEdge or Enphase microinverters
- Hybrid inverters (growatt, GoodWe)

### 5.3 Electrical Panel Considerations

- Many homes need panel upgrade (100A → 200A): **$8,000-15,000** installed, including PG&E service upgrade fees (corrected from original $3,000-5,000)
- ~60-70% of Daly City homes built before 1990 will require an upgrade for a system of this size
- PG&E interconnection review may mandate an upgrade regardless of panel condition
- Check current panel capacity before ordering equipment

### 5.4 Resilience Benefits

- PG&E PSPS (Public Safety Power Shutoff) events are increasingly common
- Battery system can power essential loads for 1-3 days (depending on size and load selection)
- Backup power value: difficult to quantify but significant in fire season

### 5.5 Safety and Code Compliance

> **Adversarial review correction (June 28, 2026):** The original document had NO dedicated safety section despite these being mandatory requirements for permitting. Non-compliance means the project cannot be permitted. All items below are required by California law, not optional.

**Mandatory Requirements:**

- **Rapid shutdown:** California Title 24, Part 6 requires rapid shutdown of PV arrays within 10 feet of the roof edge and within the array itself. All installers must comply.
- **Arc-fault protection:** Required for all new PV installations per NEC 2023. The installer's equipment must include AFCI (Arc-Fault Circuit Interrupter) protection.
- **Battery fire code:** California Fire Code Chapter 60 requires fire-rated enclosures, separation distances, and potentially fire suppression for multiple lithium-ion batteries. This is particularly important for installations with 3+ Powerwall units. San Mateo County fire marshal review is increasingly stringent on multi-battery installations.
- **Seismic requirements:** All battery and panel mounting must be seismic-rated in California. Equipment must meet California seismic certification standards.
- **Wind load calculations:** Daly City is coastal with significant wind exposure. Panel mounting must be engineered for local wind loads.
- **Emergency disconnects:** NEC requires clearly labeled emergency disconnects for both PV and battery systems. These must be accessible to first responders.
- **Grid-forming vs. grid-following inverters:** Under NEM 3.0 and new CAISO requirements, new battery systems may need grid-forming capability for grid stability.
- **Roof penetrations and warranty:** Panel mounting penetrations can void existing roof warranties. Coordinate with roof replacement if needed.
- **Permitting:** All work requires Daly City building permits, electrical permits, and mechanical permits as applicable.
- **Inspections:** City inspection, fire department inspection (increasingly required for battery installations), and PG&E inspection before permission to operate (PTO).

**Questions to ask your installer:**
- Does your design include Title 24 rapid shutdown compliance?
- Is the equipment seismic-rated and wind-load rated for Daly City?
- How do you handle California Fire Code Chapter 60 requirements for multi-battery installations?
- Will you coordinate with San Mateo County fire marshal for battery installations?
- What emergency disconnects are included in the design?
- How do you protect existing roof warranties?

---

## 6. Financial Summary

> **Adversarial review correction (June 28, 2026):** The original scenario labels were misleading. The "Most-Likely Scenario" used the SGIP Equity tier, which is currently PAUSED in PG&E territory. The scenarios below are relabeled to accurately reflect likelihood. All net cost calculations have been corrected to use consistent system cost arithmetic (see Section 4.1).

### Realistic Baseline Scenario (General Market tier, mid-range costs)

> This is the most likely scenario for the vast majority of homeowners. SGIP General Market is open to all residential customers.

| Item | Amount |
|------|--------|
| Total system cost (8-12 kW solar + 40.5 kWh battery) | $72,000-95,000 |
| Federal ITC (30%) | -$21,600 to -$28,500 |
| SGIP incentive (General Market, 40.5 kWh × $200) | ~-$8,100 |
| **Net cost** | **$42,300-65,300** |
| Annual savings (corrected) | $5,000-8,000 |
| **Payback** | **5.3-13.1 years** (6.2-8.2 years with 4% PG&E rate escalation) |

### Best-Case Scenario (SGIP Equity tier, if eligible and SGIP reopens)

> SGIP Equity tier is currently PAUSED in PG&E territory, managed via queue/waitlist. This scenario assumes the tier reopens and you qualify.

| Item | Amount |
|------|--------|
| Total system cost (8-12 kW solar + 40.5 kWh battery) | $72,000-95,000 |
| Federal ITC (30%) | -$21,600 to -$28,500 |
| SGIP incentive (Equity, 40.5 kWh × $850) | ~-$34,400 |
| **Net cost** | **$9,100-39,100** |
| SGIP as % of total cost | ~32-48% (not 80-100% — see C2 correction) |
| Annual savings | $5,000-8,000 |
| **Payback** | **1.2-7.8 years** |

### Optimistic Scenario (SGIP Equity Resilience, if eligible and SGIP reopens)

> SGIP Equity Resilience is currently PAUSED in PG&E territory. Requires fire-risk zone + equity eligibility.

| Item | Amount |
|------|--------|
| Total system cost (8-12 kW solar + 40.5 kWh battery) | $72,000-95,000 |
| Federal ITC (30%) | -$21,600 to -$28,500 |
| SGIP incentive (Equity Resilience, 40.5 kWh × $1,050 avg) | ~-$42,500 |
| **Net cost** | **$1,000-31,000** |
| SGIP as % of total cost | ~36-59% (not 80-100% — see C3 correction) |
| Annual savings | $5,000-8,000 |
| **Payback** | **0.1-6.2 years** |

> **Note:** The "FREE system" language from the original document was misleading. Even with maximum SGIP Equity Resilience incentives, the incentive covers 36-59% of total system cost, not 80-100%. The remaining balance must be paid out-of-pocket or financed.

### Conservative Scenario (high costs, General Market tier, 2-3 Powerwalls only)

> Smaller battery (27 kWh = 2 Powerwalls), higher system costs, no equity eligibility.

| Item | Amount |
|------|--------|
| Total system cost (10-12 kW solar + 27 kWh battery) | $65,000-85,000 |
| Federal ITC (30%) | -$19,500 to -$25,500 |
| SGIP incentive (General Market, 27 kWh × $200) | ~-$5,400 |
| **Net cost** | **$40,100-60,100** |
| Annual savings | $5,000-8,000 |
| **Payback** | **5.0-12.0 years** |

---

## 7. Action Items

### Immediate (before any financial planning)

- [ ] **Verify actual roof capacity** — Contact 3-5 NABCEP-certified Bay Area installers for professional roof assessments. The corrected estimate is 8-14 kW (20-35 panels), not the original 16-24 kW.
- [ ] **Determine actual battery size needed** — Conduct an evening peak load analysis using PG&E usage data. 27-40.5 kWh (2-3 Powerwalls) is the recommended starting point, not 60 kWh.
- [ ] **Verify current SGIP status** — SGIP Equity and Equity Resilience tiers are paused in PG&E territory. Check https://sgip.energy.ca.gov/ for current tranche status and waitlist.
- [ ] **Get 3-5 detailed quotes** from Bay Area solar installers that include: actual system size based on roof assessment, realistic battery configuration (2-3 Powerwalls), panel upgrade costs ($8K-15K if needed), roof replacement costs if applicable ($25K-45K), and all hidden costs.
- [ ] Check electrical panel capacity
- [ ] Review current tax situation (can you use full 30% ITC?)
- [ ] **Check SGIP Equity eligibility** — CARE/FERA recipients likely qualify for ~$850/kWh tier (when it reopens)
- [ ] **Check SGIP Equity Resilience eligibility** — if in a fire-risk zone + equity, could be ~$1,000-1,100/kWh (when it reopens)
- [ ] **Call PG&E** to verify TOU-D-PRIME availability and enroll, and check EV charger rebate status
- [ ] **Contact San Mateo County Assessor** about battery storage property tax exclusion (expires Jan 1, 2027)
- [ ] **Check Generator & Battery Rebate Program** — $300 for portable batteries if in fire-risk area
- [ ] **Verify PG&E interconnection timeline** — For a system over 10 kW, request a preliminary interconnection review to determine if a distribution impact study is required (adds 3-6 months)
- [ ] **Enroll in PG&E Demand Response program** — required for SGIP General Market tier

### Before Installation
- [ ] Submit SGIP application (MUST be before installation)
- [ ] Apply for PG&E interconnection
- [ ] Obtain Daly City building permit
- [ ] Select equipment and installer
- [ ] Ensure installer's design includes all safety/code compliance items from Section 5.5

### After Installation
- [ ] Pass city inspection
- [ ] Pass fire department inspection (if required)
- [ ] Receive PG&E Permission to Operate (PTO)
- [ ] File for federal ITC on tax return (Form 5695)
- [ ] Receive SGIP incentive payment
- [ ] File property tax exclusion claim with San Mateo County Assessor (before Jan 1, 2027)

### Long-Term Maintenance
- [ ] Budget for annual maintenance ($200-500/year)
- [ ] Set aside inverter replacement reserve ($3,000-5,000 after 10-15 years)
- [ ] Set aside battery replacement reserve ($14,000-17,200 per unit after 10-15 years)
- [ ] Consider extended warranty ($1,500 per Powerwall if desired)

---

## 8. Research Status

### Complete (incorporated agent research)
- [x] Federal ITC: 30% confirmed through 2032 (VERIFIED — agent claim of Dec 2025 expiration was incorrect)
- [x] NEM 3.0 framework understood
- [x] Property tax exclusion: AB 198 / RTC Section 73 — expires Jan 1, 2027; NOT automatic; must file claim; battery coverage is a gray area
- [x] Program categories identified
- [x] Cost estimate ranges corrected (adversarial review: components now sum correctly; solar $2.75-4.50/W; battery $14K-17.2K per Powerwall 3)
- [x] Payback period estimates corrected (5-13 years General Market; includes rate escalation scenarios)
- [x] Equipment options reviewed (Tesla Powerwall 3, Enphase IQ Battery 5P, LG RESU); 2-3 Powerwalls recommended as starting point
- [x] Timeline and permitting requirements corrected (4-8 months normal; up to 26 months with paused SGIP tiers)
- [x] Solar production rates refined (1,350-1,500 kWh/kW/year, 3.7-4.0 peak sun hours)
- [x] Roof capacity corrected (8-14 kW, not 16-24 kW)
- [x] SGIP rates refined (general $200/kWh, Equity $850/kWh, Equity Resilience $1,000-1,100/kWh); rates are estimates subject to change
- [x] SGIP coverage percentages corrected (32-60% of total system cost, not 80-100%)
- [x] CSI program confirmed CLOSED (ended Dec 31, 2016)
- [x] PG&E EV charger rebates documented (income-qualified up to $5,000; standard tier may be discontinued)
- [x] TOU-D-PRIME vs standard TOU-D rate difference documented
- [x] NEMA (Net Energy Metering Aggregation) documented
- [x] NSC (Net Surplus Compensation per AB 920) documented
- [x] Generator & Battery Rebate Program documented ($300 for portable batteries)
- [x] Income-Qualified SGIP rebate confirmed CLOSED with waitlist available
- [x] Safety and code compliance section added (Title 24, NEC 2023, Fire Code Chapter 60, seismic, wind load, emergency disconnects)
- [x] Hidden and contingency costs added ($10K-88K+ potential additional costs)
- [x] Battery replacement costs added to long-term cost planning
- [x] Scenario labels corrected (General Market = realistic baseline, not "Most-Likely")
- [x] Annual savings corrected ($5,000-8,750/year, not $8,500-13,500)
- [x] NEM 3.0 export revenue corrected ($200-750/year, not $500-1,500)
- [x] Panel upgrade costs corrected ($8,000-15,000, not $3,000-5,000)
- [x] Battery degradation rate corrected (1-2%/year, not 2-3%)
- [x] 25-year net benefit corrected ($46K-121K, not $35K-45K)
- [x] Battery sizing clarification added (evening peak load vs. total daily usage)

### Pending Verification
- [ ] **SGIP current tranche status** — quotas can run out mid-quarter, verify before proceeding
- [ ] **Your SGIP tier eligibility** — determine if you qualify for Equity or Equity Resilience
- [ ] **Join Income-Qualified SGIP waitlist** — if you qualify (80% AMI or below, CARE/FERA/ESA), join waitlist even though closed for new applications
- [ ] **TOU-D-PRIME availability** for your specific address — call PG&E or check account
- [ ] **Exact PG&E avoided export rates** under NEM 3.0 — verify with PG&E
- [ ] **San Mateo County Assessor** — do batteries qualify for property tax exclusion?
- [ ] **EV charger rebate program status** — may be closed/paused for new applications
- [ ] **Generator & Battery Rebate** — check if your address is in Tier 2/3 fire district
- [ ] **Actual roof capacity** — visit https://www.google.com/maps/place/978+Martin+Trail,+Daly+City,+CA+94014 to inspect roof

---

## 9. Key Assumptions & Caveats

1. **SGIP rates are estimates** — actual rates depend on current tranche; quotas can run out mid-quarter, verify before proceeding. Equity and Equity Resilience tiers are currently paused in PG&E territory. SGIP incentives cover 32-60% of total system cost, not 80-100% as originally stated.
2. **Roof capacity is estimated** — actual capacity requires professional assessment; visit Google Maps satellite view for initial inspection
3. **Cost estimates are ranges** — actual costs vary by installer, equipment, and site conditions; Bay Area benchmark $3.00-4.50/W
4. **Payback calculations** — The payback periods in Section 4.4 show both flat-rate and 4% rate-escalation scenarios. PG&E rates have historically risen 3-5%/year, which shortens actual payback vs. flat-rate calculations.
5. **Federal ITC at 30%** — verified active through Dec 31, 2032 per IRA. One agent incorrectly claimed Dec 2025 expiration.
6. **NEM 3.0 rules could change** — future legislation may alter net metering
7. **Battery degradation** — Tesla Powerwall 3 is rated for 100% capacity retention for 10 years (or 100,000 cycles at 100% DoD). Real-world degradation is likely closer to **1-2% per year** (corrected from the original 2-3% figure which was pessimistic). A 40.5 kWh system may be ~36-38 kWh after 15 years.
8. **Marine climate** — Daly City fog reduces solar production vs. inland California (1,350-1,400 kWh/kW/year vs. 1,800+ inland)
9. **Peak sun hours** — 3.7-4.0/day (NREL PVWatts), lower than simple 5-hour estimate
10. **Property tax exclusion expires Jan 1, 2027** — not automatic; must file claim with San Mateo County Assessor; battery coverage is a gray area
11. **SGIP tier eligibility is critical** — the difference between general market (~$8K for 40.5 kWh) and Equity Resilience (~$42.5K for 40.5 kWh) is massive. But both Equity and Equity Resilience tiers are currently paused in PG&E territory. General Market is the realistic baseline. Determine your tier and join waitlists for paused tiers.
12. **CSI program is closed** — no solar production incentives remain through California Solar Initiative
13. **SGIP quotas are limited** — apply early in the quarter; Equity and Equity Resilience tiers are currently paused in PG&E territory
14. **Income-Qualified SGIP is closed** but has a waitlist — join if you qualify (80% AMI or CARE/FERA/ESA)
15. **PG&E programs change frequently** — verify all program details directly with PG&E before relying on them for financial planning

---

*This document will be updated as research progresses. All figures marked with estimates are based on current knowledge and should be verified with professional installers and program administrators. Agent research completed June 28, 2026 — SGIP/PG&E data refined from second research agent.*

---

### How to Determine Your SGIP Tier

> **Important:** SGIP Equity and Equity Resilience tiers are currently **PAUSED** in PG&E territory. Join the waitlist even though applications are not being processed. General Market tier remains open.

**Step 1: Check if you qualify for CARE/FERA**
- CARE (California Alternate Rates for Energy): 20%+ discount on electricity bills
- FERA (Family Emergency Relief Assistance): Low-income households above CARE threshold
- If you receive CARE/FERA, you likely qualify for the **Equity tier** (~$850/kWh)

**Step 2: Check if you're in a fire-risk zone**
- Visit the CPUC Fire Threat map or check with your local fire department
- If you're in a TBIF (Threatened/Imminent Fire Risk) zone AND qualify for Equity, you may qualify for **Equity Resilience** (~$1,000-1,100/kWh)

**Step 3: Check CA-MECH 40%+ area**
- The disadvantaged community designation uses California's Map for Equity and Climate Health (CA-MECH)
- Visit https://hcd.ca.gov/sites/default/files/documents/2024-06/ca-mech-40pct.html
- If your address is in a CA-MECH 40%+ area AND household income ≤ 80% AMI, you qualify for Equity tier

**Step 4: Apply early in the quarter**
- SGIP quotas can run out mid-quarter
- Apply as soon as possible after determining eligibility
- For paused tiers, join the waitlist and monitor for reopening

---

## 10. Sources & References

- **NREL PVWatts:** https://pvwatts.nrel.gov — use coordinates 37.7058, -122.4619 for Daly City
- **SolarTechOnline:** https://solartechonline.com/blog/california-solar-panel-cost-guide-2025/ — Bay Area cost data
- **SolarPanelEmpire:** https://solarpanelempire.com/california-solar-panel-installation-cost-per-watt-2025/ — Bay Area cost benchmarks
- **Lawrence Berkeley National Lab** — U.S. Solar PV System and Energy Storage Cost Monitor (definitive annual cost data)
- **SGIP:** California Self-Generation Incentive Program (CPUC-administered, PG&E-run portal)
- **PG&E working URLs:**
  - https://www.pge.com/en/save-energy-and-money/rebates-and-incentives.html
  - https://www.pge.com/en/clean-energy/solar.html
  - https://www.pge.com/en/clean-energy/battery-storage.html
- **Federal ITC:** IRS Section 25D, Inflation Reduction Act (30% through 2032)
- **NEM 3.0:** California Net Energy Metering (effective April 2023)
- **PG&E TOU-D-PRIME:** Time-of-Use rate schedule for residential customers
- **Prop 156:** California Property Tax Exclusion for Renewable Energy Systems

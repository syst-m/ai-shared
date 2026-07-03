# Adversarial Review — solar-installation.md

**Reviewed by:** Three independent analysts (Financial, Regulatory, Practical)
**Document under review:** solar-installation.md (978 Martin Trail, Daly City, CA 94014)
**Review date:** June 27, 2026

---

## 1. Executive Summary

- **The document's headline figures are unreliable for decision-making.** The total system cost ($85,000-$130,000) does not reconcile with its own component breakdown ($94,000-$160,000). The SGIP Equity and Equity Resilience coverage percentages (80-100%) are overstated by a factor of 1.5-2x when measured against total installed system cost; actual ranges are 39-60% and 46-78%.
- **The "Most-Likely Scenario" is mislabeled.** It assumes the highest available SGIP tier (Equity, $51,000 incentive), which is currently paused and waitlisted. The General Market scenario ($61,500 net cost, 6.2-year payback) is the realistic baseline for the vast majority of homeowners.
- **System size estimates are physically unrealistic.** The document claims 16-24 kW (40-60 panels) can fit on a typical Daly City home, but a professional installer would estimate 8-14 kW (20-35 panels) based on actual usable roof area after setbacks, vents, and obstructions. The recommended 4-5 Tesla Powerwall 3 units (60 kWh) is impractical for space, panel capacity, and fire code compliance.
- **Annual savings figures are inflated.** The stated $8,000-12,000/year exceeds the household's current electricity bill ($6,000-8,400/year) without sufficient justification. Under NEM 3.0 with realistic export rates, actual savings are more likely $5,000-8,000/year.
- **Critical safety and code compliance topics are entirely missing.** Rapid shutdown requirements, arc-fault protection, California Fire Code Chapter 60 battery requirements, seismic and wind load engineering, and emergency disconnects are not addressed despite being mandatory for permitting.

---

## 2. Critical Issues (must fix before relying on document)

### C1. Total system cost arithmetic does not add up

**Location:** Lines 288-295 (Section 4.1) and lines 334-338 (Section 4.2)

**Problem:** The component line items sum to $94,000-$160,000:
- Solar panels: $28,000-54,000
- Battery: $56,000-86,000
- Inverter/electrical: $3,000-6,000
- Permitting/engineering: $2,000-4,000
- Soft costs: $5,000-10,000
- **Sum: $94,000-$160,000**

But the stated total is $85,000-$130,000. The $85,000 low end is $9,000 below the minimum sum of components. The $130,000 high end is $30,000 below the maximum. All downstream net cost and payback calculations (lines 329-332, lines 410-452) are affected.

**Fix:** Either correct the component line items to sum to $85,000-$130,000, or correct the stated total to $94,000-$160,000. Recalculate all net cost and payback figures accordingly.

---

### C2. SGIP Equity tier coverage claim is materially overstated

**Location:** Lines 68-69 (Section 2.2 table), lines 322-323 (Section 4.2), lines 421-430 (Section 6)

**Problem:** Section 2.2 describes the Equity tier as covering "80%-100% of system cost." At $850/kWh for a 60 kWh battery, the SGIP Equity incentive is $51,000. Against the stated total system cost of $85,000-$130,000, this covers only 39-60% of total cost, not 80-100%. The 80-100% figure only applies to the battery-subsystem cost ($56,000-86,000), not the full installed system.

**Fix:** Replace the "80%-100% of system cost" framing with "approximately 59-91% of battery cost, 39-60% of total installed system cost." Update Section 6 scenario tables to reflect correct net costs.

---

### C3. SGIP Equity Resilience tier coverage claim is materially overstated

**Location:** Lines 69 (Section 2.2 table), lines 323 (Section 4.2), lines 410-419 (Section 6)

**Problem:** The document states Equity Resilience covers "80%-100% of system cost." At $1,000-1,100/kWh for 60 kWh, the incentive is $60,000-$66,000. Against the stated total system cost of $85,000-$130,000, this covers 46-78% of total cost, not 80-100%.

**Fix:** Replace the "80%-100% of system cost" framing with "approximately 46-78% of total installed system cost." Update Section 6 to remove "FREE" and "essentially FREE" language unless the math truly supports negative net cost after all components.

---

### C4. Roof capacity estimate is physically unrealistic

**Location:** Lines 230-235 (Section 3.1)

**Problem:** The document states "Panels that could fit: 40-60 panels / Estimated solar capacity: 16-24 kW DC." For a typical 1,500-2,500 sq ft Daly City home with 800-1,200 sq ft total roof area (not usable area), after accounting for setbacks (18-24 inch clearance on all sides for fire access), ridge vents, plumbing vents, skylights, and HVAC equipment, the usable area is closer to 400-700 sq ft. This fits 20-35 panels maximum, yielding 8-14 kW DC -- not 16-24 kW. Getting to 60 panels would require 1,200 sq ft of clear, unshaded roof, which does not exist on any real Daly City tract home.

**Fix:** Replace the 16-24 kW range with 8-14 kW DC (20-35 panels). Update production estimates in Section 3.2 accordingly. The 8 kW system producing 10,800-12,000 kWh/year (line 256) is already more than adequate for the stated usage.

---

### C5. 4-5 Tesla Powerwall 3 units is impractical

**Location:** Lines 18, 380 (Section 1 and Section 5.2)

**Problem:** Five Powerwall 3 units require nearly 13 linear feet of wall space side-by-side, or roughly 7.5 feet wide by 8.3 feet tall stacked 2-high. This is a significant footprint most Daly City homes do not have. Each unit has 11.5 kW continuous output; five units can deliver 57.5 kW of backup power, far exceeding what a standard residential panel can handle for backup distribution. Multiple large lithium-ion batteries trigger California Fire Code Chapter 60 requirements including fire-rated separation, specific mounting requirements, and potentially a dedicated battery room. San Mateo County fire marshal review is increasingly stringent on multi-battery installations.

**Fix:** Recommend 2-3 Powerwall 3 units (27-40.5 kWh) as the realistic starting point, with the option to add more later. Add a dedicated section on fire code requirements and physical space constraints for multi-battery installations.

---

### C6. Missing safety and code compliance topics

**Location:** Entire document has no dedicated safety section

**Problem:** The document omits several mandatory requirements:
- **Rapid shutdown:** California Title 24, Part 6 requires rapid shutdown of PV arrays within 10 feet of the roof edge and within the array itself.
- **Arc-fault protection:** Required for all new PV installations per NEC 2023.
- **Battery fire code:** California Fire Code Chapter 60 requires fire-rated enclosures, separation distances, and potentially fire suppression for multiple lithium-ion batteries.
- **Seismic requirements:** All battery and panel mounting must be seismic-rated in California.
- **Wind load calculations:** Daly City is coastal with significant wind exposure.
- **Emergency disconnects:** NEC requires clearly labeled emergency disconnects for both PV and battery systems.
- **Grid-forming vs. grid-following inverters:** Under NEM 3.0 and new CAISO requirements, new battery systems may need grid-forming capability.
- **Roof penetrations and warranty:** Panel mounting penetrations can void existing roof warranties.

**Fix:** Add a dedicated section (after Section 5.4) covering all of the above. These are not optional -- non-compliance means the project cannot be permitted.

---

### C7. Hidden costs totaling $36,000-88,000+ not included

**Location:** Section 4 (lines 282-357)

**Problem:** The document's cost estimates omit several significant costs:
- **Roof replacement or reinforcement:** $25,000-45,000 if the existing roof is older than 10 years (common for 1960s-70s Daly City homes).
- **PG&E interconnection application fee:** $1,000-2,000.
- **Utility infrastructure upgrades:** $5,000-25,000 if PG&E determines the local transformer or distribution line needs upgrading.
- **Impact fees:** $1,000-5,000 for connection or development fees.
- **Structural engineering report:** $500-2,000.
- **Tree trimming/removal:** $500-3,000 per tree.
- **Extended battery warranty:** ~$1,500 per Powerwall; $6,000-7,500 for 4-5 units.
- **Homeowner's insurance increase:** $100-300/year.
- **Sales/use tax on non-exempt components:** ~1-2% of equipment costs.

**Fix:** Add a "Contingency and Hidden Costs" line item of $10,000-25,000 to the cost table (line 295). Add specific line items for roof replacement ($25,000-45,000 if needed), PG&E infrastructure upgrades ($5,000-25,000 if needed), impact fees ($1,000-5,000), and extended warranties ($1,500-7,500).

---

### C8. "Most-Likely Scenario" is actually the best-case scenario

**Location:** Lines 421-430 (Section 6)

**Problem:** The document labels the Equity tier scenario (Equity + mid-range costs, $22,500 net cost, 2.3-year payback) as "Most-Likely." This assumes the CARE/FERA SGIP Equity tier ($850/kWh, $51,000 incentive), which is the highest tier available to most income-qualified customers. The SGIP Equity and Equity Resilience tiers are currently PAUSED in PG&E territory (lines 68-69). The General Market scenario ($61,500 net cost, 6.2-year payback) is the realistic baseline for the vast majority of homeowners.

**Fix:** Relabel the scenarios: General Market = "Realistic Baseline," Equity tier = "Best Case (if eligible and SGIP reopens)," Equity Resilience = "Best Case (if eligible and SGIP reopens)." Move the Equity Resilience scenario last as "Optimistic (conditional)."

---

## 3. Major Issues (should fix for accuracy)

### M1. Annual savings exceed current electricity costs without adequate justification

**Location:** Lines 15-16 (Section 1), lines 344-347 (Section 4.3)

**Problem:** The document states annual energy savings of $8,000-12,000/year (line 345) but the household's current monthly electricity cost is $500-700, or $6,000-8,400/year (lines 15-16). For savings to exceed current costs, the system must produce surplus energy exported at NEM 3.0 rates. The document lists NEM 3.0 export revenue as only $500-1,500/year (line 346). Even adding the high end: $6,000 (current bill fully offset) + $1,500 (export) = $7,500, which is below the stated $8,000 minimum. The savings figures appear inflated or double-counted.

**Fix:** Recalculate savings assuming the system offsets current usage ($6,000-8,400/year) plus realistic export revenue ($500-750/year based on lines 3-6 review). Realistic total annual savings under NEM 3.0 with a 10-14 kW system and 27-40 kWh battery: $5,000-8,000/year (General Market tier).

---

### M2. Payback period note contradicts stated rate increase assumption

**Location:** Line 351 (Section 4.3 notes) vs. line 527 (Section 9 caveat #4)

**Problem:** Line 351 states "PG&E rates are rising ~3-5%/year, which shortens payback over time." But caveat #4 states "Payback assumes no rate increases." These are directly contradictory. If rates rise 3-5%/year, the payback period is materially shorter than stated. The 6.2-year payback in the General Market scenario would be closer to 5-5.5 years with 4% annual rate escalation.

**Fix:** Remove caveat #4 or rephrase to "Payback calculations shown use flat rates for simplicity; actual payback is shorter with PG&E rate escalations of 3-5%/year." Ensure the payback calculations in Section 6 incorporate rate escalation where line 351 claims they do.

---

### M3. 25-year savings figure is internally inconsistent with stated annual savings

**Location:** Line 355 (Section 4.3 notes)

**Problem:** The document states "25-year savings: $35,000-45,000 for solar alone." But it also states annual energy savings of $8,000-12,000/year (line 345). At even the low end of $8,000/year, 25 years of savings = $200,000. The stated $35,000-45,000 figure is roughly 5 years of savings, not 25. This figure does not represent cumulative savings. It may represent net savings after costs, but it is not labeled as such.

**Fix:** Clarify whether this figure represents cumulative gross savings, net savings after system costs, or something else. If it is net savings after costs, relabel it as "25-year net benefit after system costs." A more accurate 25-year cumulative savings figure for the General Market scenario would be approximately $125,000-$200,000 (before discounting), minus system costs of $61,500-$79,000, yielding $46,000-$121,000 net benefit.

---

### M4. Battery cycling savings assumption is unrealistic

**Location:** Lines 157-158 (Section 2.5), lines 275-276 (Section 3.3)

**Problem:** Line 157 states "Savings per cycle: ~$0.35/kWh x 60 kWh = ~$21/day from cycling alone." This assumes a full 60 kWh charge-discharge cycle every single day. At 90% round-trip efficiency, 60 kWh of stored energy delivers 54 kWh of discharge. The savings from TOU arbitrage on 54 kWh at $0.35/kWh is $18.90/day, not $21/day. Line 276 claims annual battery value of $5,500-9,000/year, implying 200-250+ full cycles per year. In Daly City's marine climate with seasonal fog and variable demand, daily full cycling is not achievable year-round. A more realistic assumption would be 200-250 full cycles per year, yielding $3,000-4,700/year from cycling alone.

**Fix:** Correct the per-cycle savings to $18.90/day (54 kWh x $0.35/kWh). Reduce the annual battery value estimate to $3,000-5,500/year. Add a caveat about seasonal variation in cycling frequency.

---

### M5. NEM 3.0 export revenue estimate exceeds what the math supports

**Location:** Lines 346 (Section 4.3), lines 113-116 (Section 2.3), lines 252-260 (Section 3.2)

**Problem:** The document states NEM 3.0 export revenue of $500-1,500/year (line 346). For an 18-20 kW system producing 24,300-30,000 kWh/year (lines 259-260) against 23,725 kWh usage (line 17), excess export is roughly 3,275-6,275 kWh. At the stated avoided rates of $0.05-0.12/kWh (lines 114-115), the maximum export revenue is 6,275 x $0.12 = $753/year. The $1,500 upper bound is not supported by the stated production and rate assumptions.

**Fix:** Reduce the NEM 3.0 export revenue estimate to $200-750/year based on the document's own production and rate assumptions.

---

### M6. Panel upgrade costs are underestimated

**Location:** Line 396 (Section 5.3)

**Problem:** The document states "Many homes need panel upgrade (100A to 200A): ~$3,000-5,000." A realistic installed cost is $8,000-15,000, depending on whether PG&E needs to upgrade the service drop. PG&E's service upgrade fees alone can add $3,000-8,000. Approximately 60-70% of Daly City homes built before 1990 will require an upgrade for a system of this size. PG&E interconnection review may mandate an upgrade regardless of panel condition.

**Fix:** Replace with "Panel upgrade (100A to 200A): $8,000-15,000 installed, including PG&E service upgrade fees. Approximately 60-70% of Daly City homes built before 1990 will require an upgrade. PG&E interconnection review may mandate an upgrade regardless of panel condition."

---

### M7. Installation timeline is optimistic

**Location:** Lines 364-373 (Section 5.1)

**Problem:** The document states a total timeline of 2-4 months. This is optimistic for a system of this complexity. Missing factors:
- **Roof condition:** If the roof needs replacement, this adds 2-8 weeks for replacement and re-permitting.
- **Structural engineering:** Required in San Mateo County for larger systems; adds 2-3 weeks.
- **PG&E interconnection:** For systems over 10 kW, the distribution impact study can take 3-6 months.
- **SGIP application:** Equity and Equity Resilience tiers are paused; waitlist could be 12-24 months.
- **Fire department inspection:** Battery installations increasingly require it; adds 2-4 weeks.
- **Utility meter upgrade:** Can take 4-8 weeks.

**Fix:** Replace with "Total timeline: 4-8 months under normal conditions. Add 2-8 weeks if roof replacement is needed. Add 3-6 months for PG&E distribution impact study on systems over 10 kW. Add 12-24 months if pursuing SGIP Equity or Equity Resilience tiers (currently paused/waitlisted)."

---

### M8. 60 kWh battery cannot cover stated daily usage

**Location:** Lines 15 (Section 1), lines 380 (Section 5.2)

**Problem:** The document states daily energy use of 50-80 kWh (line 15) and plans a 60 kWh battery (line 18). Even at 100% round-trip efficiency (physically impossible), a 60 kWh battery cannot cover 80 kWh of daily usage. At realistic 90% efficiency, a 60 kWh battery delivers approximately 54 kWh. This covers the low end (50 kWh) but leaves a 26 kWh gap at the high end. The document conflates total daily usage with evening peak demand. A household using 65 kWh/day may have a peak evening load (4-9 PM) of only 20-30 kWh, which a 40-60 kWh battery could cover.

**Fix:** Add a clarification: "A 40-60 kWh battery system can cover typical evening peak loads (20-40 kWh) for most of the night. It will not cover 100% of daily usage (50-80 kWh), but it can cover the majority of evening peak demand when electricity rates are highest. The battery should be sized based on evening peak load, not total daily usage."

---

### M9. Battery replacement costs entirely omitted

**Location:** Section 4 (lines 282-357), Section 9 (line 530)

**Problem:** The document mentions battery degradation (line 530: "batteries lose ~2-3% capacity per year") but has no discussion of replacement costs. Tesla Powerwall 3 has a 10-year warranty (extendable to 25 years for ~$1,500). After warranty, replacement cost is approximately $14,000-17,200 per unit. For a 5-unit system, that is $70,000-86,000 in replacement batteries alone -- roughly 80-90% of the original cost. This is a massive hidden cost that the document completely omits. Inverter replacement ($1,500-3,000 after 10-15 years) is also not mentioned.

**Fix:** Add a "Long-Term Maintenance and Replacement Costs" section covering: annual inspection ($200-500/year), inverter replacement reserve ($3,000-5,000 after 10-15 years), battery replacement reserve ($14,000-17,200 per unit after 10-15 years), and a 25-year total cost of ownership calculation that includes these replacements.

---

### M10. SGIP rates compared against wrong cost base

**Location:** Lines 421-430 (Section 6)

**Problem:** The SGIP rates ($200, $850, $1,000-1,100 per kWh) are presented as percentages of battery cost. But financial decisions should be evaluated against total installed system cost ($1,567-2,667/kW for the full system). SGIP Equity at $850/kWh is 59-91% of battery cost but only 39-60% of total system cost. This framing inflates the perceived value of incentives.

**Fix:** Add a column to the scenario tables showing SGIP incentive as a percentage of total system cost, not just battery cost.

---

## 4. Minor Issues (nice to fix)

### MN1. SGIP incentive calculations based on 60 kWh but actual configuration is 54 or 67.5 kWh

**Location:** Lines 18, 72-75 (Section 1, Section 2.2)

**Problem:** The document states "Planned battery capacity ~60 kWh (likely 4-5x Tesla Powerwall 3 at 13.5 kWh each)." But 4 x 13.5 = 54 kWh and 5 x 13.5 = 67.5 kWh. No configuration yields exactly 60 kWh. All SGIP incentive calculations use 60 kWh as the basis. If 4 Powerwalls (54 kWh) are used, the General Market SGIP incentive is $10,800 not $12,000. This affects the net cost in every scenario by $1,200.

**Fix:** Present calculations for both 54 kWh (4 Powerwalls) and 67.5 kWh (5 Powerwalls) configurations, or use the exact kWh values in all SGIP calculations.

---

### MN2. SGIP Equity tier rate may be inflated for 2026

**Location:** Lines 68-69, 322-323 (Section 2.2, Section 4.2)

**Problem:** The Equity rate of $850/kWh and Equity Resilience rate of $1,000-1,100/kWh appear at the top end of historical SGIP rates. SGIP funding tranches have been depleting, and rates decline over time. These rates may not be achievable in the current tranche. The document flags SGIP rates as estimates (caveat #1, line 524) but presents them as verified research (Section 8).

**Fix:** Add a stronger caveat that SGIP rates are subject to change and may be lower in future tranches. Recommend verifying current rates with PG&E before relying on specific dollar amounts.

---

### MN3. Peak sun hours explanation has a unit error

**Location:** Line 263 (Section 3.2)

**Problem:** The document states "The earlier estimate of 1,825 kWh/kW was based on a simple 5 hours x 365 calculation." The unit "kWh/kW" is a capacity factor, not an energy yield. The correct unit is kWh/kW/year or simply kWh per kW installed.

**Fix:** Change "1,825 kWh/kW" to "1,825 kWh/kW/year" or "1,825 kWh per kW installed per year."

---

### MN4. Property tax exclusion treatment is inconsistent

**Location:** Line 135 (Section 2.4) vs. line 324 (Section 4.2)

**Problem:** Line 135 explicitly states "Battery storage is a gray area -- The exclusion specifically covers 'active solar energy systems.' Batteries are not explicitly listed." But line 324 lists "Property tax exclusion" as a definite "Ongoing savings" in the incentives table without any caveat. If batteries are excluded, the property tax savings are materially lower.

**Fix:** Add a caveat to the incentives table (line 324) noting that property tax exclusion applies to solar components but battery coverage is uncertain and depends on county assessor interpretation.

---

### MN5. Battery degradation rate is aggressive

**Location:** Line 530 (Section 9 caveat #7)

**Problem:** The document states batteries lose "2-3% capacity per year." Tesla Powerwall 3 is rated for 100% capacity retention for 10 years (or 100,000 cycles at 100% DoD, or 365,000 cycles at partial DoD). Real-world degradation is closer to 1-2% per year. The 2-3% figure is pessimistic rather than wrong but skews the long-term analysis.

**Fix:** Clarify that the 2-3% figure is a conservative estimate and that real-world degradation for Powerwall 3 is likely closer to 1-2% per year based on manufacturer specifications.

---

### MN6. Generator & Battery rebate CARE/FERA tier removed but not all references updated

**Location:** Line 194 (Section 2.5.5)

**Problem:** The document correctly shows "$300 per qualified customer" (Reviewer 2 fix). However, Reviewer 2 confirmed removal of the "$500 if CARE/FERA" claim. Verify no stale references remain.

**Fix:** Confirm with grep that no references to "$500 CARE/FERA" tier remain in the document.

---

## 5. Internal Contradictions

### Contradiction 1: Payback rate increase assumption

**Location:** Line 351 vs. line 527

- Line 351 (Section 4.3 notes): "PG&E rates are rising ~3-5%/year, which shortens payback over time"
- Line 527 (Section 9 caveat #4): "Payback assumes no rate increases"

These two statements are directly contradictory. If rates rise 3-5%/year, the payback period is materially shorter than stated. The 6.2-year payback in the General Market scenario would be closer to 5-5.5 years with 4% annual rate escalation.

### Contradiction 2: SGIP tier status vs. scenario treatment

**Location:** Lines 68-69 vs. lines 421-430

- Lines 68-69 (Section 2.2): SGIP Equity and Equity Resilience tiers are **PAUSED** in PG&E territory, managed via queue/waitlist.
- Lines 421-430 (Section 6): The "Most-Likely Scenario" assumes the Equity tier ($51,000 incentive) as if it is currently available.

A paused tier with a waitlist cannot be the basis for a "most likely" scenario.

### Contradiction 3: SGIP demand response requirement

**Location:** Line 93 vs. line 506

- Line 93 (Section 2.2): "The 52-discharge-per-year requirement applies to commercial DR programs, not the residential General Market tier"
- Line 506 (Section 8): "SGIP General Market -- demand response enrollment is NOT a stated requirement for residential tier"

These are consistent (both correctly state that DR is not required for residential), but the original document may have previously stated otherwise. Verify no contradictory language remains.

### Contradiction 4: System cost vs. component sum

**Location:** Lines 288-295 vs. line 295

- Lines 288-294: Component costs sum to $94,000-$160,000
- Line 295: Stated total is $85,000-$130,000

The total does not reconcile with the components it is composed of.

---

## 6. Confidence Assessment

### HIGH CONFIDENCE (reliable sections)

- **Section 2.1 Federal ITC (lines 32-51):** Accurate and verified. 30% through 2032, phase-down schedule correct.
- **Section 2.3 NEM 3.0 (lines 98-117):** Framework is correct. Export rates and battery necessity are accurately described. The avoided rate approximation note (line 113) has been corrected to note seasonal variation.
- **Section 2.4 Property Tax Exclusion (lines 120-137):** AB 198 / RTC Section 73 description is accurate. The battery gray area caveat is appropriately stated.
- **Section 2.5 PG&E TOU-D-PRIME (lines 142-159):** Rate structure and battery value mechanism are correctly explained.
- **Section 8 Research Status (lines 486-507):** Generally accurate and well-documented.

### MODERATE CONFIDENCE (needs verification)

- **Section 2.2 SGIP rates (lines 54-95):** Rates are estimates that may change with funding tranches. Equity and Equity Resilience tiers are paused. The specific dollar amounts ($850/kWh, $1,000-1,100/kWh) should be verified with PG&E before relying on them.
- **Section 4 Cost estimates (lines 282-357):** Component costs are based on industry benchmarks but the arithmetic does not reconcile. Actual costs will vary by installer and site conditions.
- **Section 3.2 Production estimates (lines 242-263):** NREL PVWatts-derived rates (1,350-1,500 kWh/kW/year) are reasonable for Daly City, but the roof capacity assumptions (lines 232-235) are unreliable.

### LOW CONFIDENCE (speculative or inaccurate)

- **Section 3.1 Roof capacity (lines 224-240):** The 16-24 kW estimate is physically unrealistic for typical Daly City homes. Actual capacity is likely 8-14 kW.
- **Section 4.3 Payback period (lines 340-357):** Based on inflated savings figures and inconsistent cost arithmetic. The 25-year savings figure ($35,000-45,000) is internally inconsistent.
- **Section 6 Financial scenarios (lines 408-453):** The "Most-Likely" scenario uses a paused SGIP tier. The "FREE system" language is misleading. Net costs and payback periods are affected by the cost arithmetic error.
- **Section 5.2 Battery options (lines 376-392):** The 4-5 Powerwall 3 configuration is impractical for space, panel capacity, and fire code compliance.
- **Section 5.1 Timeline (lines 362-373):** 2-4 months is optimistic by 2-4x for this complexity.

---

## 7. Recommended Next Steps

### Immediate (before any financial planning)

1. **Verify actual roof capacity.** Obtain a professional roof assessment (structural, condition, shading) from a licensed Daly City solar installer. The 16-24 kW estimate should be replaced with the actual measured capacity, which is likely 8-14 kW.
   - Action: Contact 3-5 NABCEP-certified Bay Area installers for roof assessments and quotes.

2. **Determine actual battery size needed.** Based on the corrected roof capacity and evening peak load analysis (not total daily usage), determine whether 27-40 kWh (2-3 Powerwalls) is sufficient, or whether a different battery system (e.g., Generac PWRcell) is more appropriate.
   - Action: Conduct an evening peak load analysis using PG&E usage data.

3. **Verify current SGIP status.** SGIP Equity and Equity Resilience tiers are paused in PG&E territory. Contact the SGIP portal or PG&E to confirm current availability and waitlist status before including these in any financial planning.
   - Action: Check https://sgip.energy.ca.gov/ for current tranche status.

### Short-term (within 30 days)

4. **Obtain actual quotes.** Get 3-5 detailed quotes from Bay Area solar installers that include:
   - Actual system size based on roof assessment
   - Realistic battery configuration (space constraints, fire code)
   - Panel upgrade costs ($8,000-15,000 if needed)
   - Roof replacement costs if applicable ($25,000-45,000)
   - All hidden costs (interconnection fees, impact fees, structural engineering)

5. **Verify PG&E interconnection timeline.** For a system over 10 kW, request a preliminary interconnection review to determine if a distribution impact study is required. This can add 3-6 months to the timeline.
   - Action: Contact PG&E interconnection department with preliminary system specs.

6. **Confirm SGIP tier eligibility.** Determine CARE/FERA status, CA-MECH 40%+ area designation, and fire-risk zone status. If qualifying for Equity or Equity Resilience, join the waitlist even though paused.
   - Action: Check CA-MECH map (line 559), PG&E fire threat map, and CARE/FERA status.

### Medium-term (before installation)

7. **Address safety and code compliance.** Ensure the installer's design includes:
   - Title 24 rapid shutdown compliance
   - NEC 2023 arc-fault protection
   - California Fire Code Chapter 60 battery requirements
   - Seismic and wind load engineering
   - Emergency disconnects
   - Roof warranty protection

8. **Plan for long-term costs.** Budget for:
   - Annual maintenance ($200-500/year)
   - Inverter replacement reserve ($3,000-5,000 after 10-15 years)
   - Battery replacement reserve ($14,000-17,200 per unit after 10-15 years)
   - Extended warranty ($1,500 per Powerwall if desired)

9. **Recalculate financial projections.** Using actual system size, actual costs (including hidden costs), realistic savings ($5,000-8,000/year General Market), and corrected SGIP incentives, recalculate payback periods and 25-year net benefits.

---

*This review synthesizes findings from three independent analysts. All line references are to the document at /Users/frankellis/ai_scratch/solar/solar-installation.md as of the review date. The document requires significant revision before being relied upon for financial or installation decisions.*

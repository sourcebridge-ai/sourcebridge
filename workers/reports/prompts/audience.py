# SPDX-License-Identifier: AGPL-3.0-or-later
# Copyright (C) 2026 SourceBridge Contributors

"""Audience-specific prompt instructions.

Each audience preset controls language, depth, framing, and how
recommendations and metrics are presented. The same data produces
fundamentally different output for different audiences.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass
class AudienceConfig:
    key: str
    title: str
    language: str
    depth: str
    recommendations: str
    metrics: str
    sample: str  # 2-sentence sample for the wizard audience card


AUDIENCES: dict[str, AudienceConfig] = {
    "c_suite": AudienceConfig(
        key="c_suite",
        title="C-Suite / Board",
        language=(
            "Write in plain business English. No technical jargon — if you must use "
            "a technical term, explain it in one sentence. Frame everything in terms "
            "of business impact: risk, cost, liability, reputation, and timeline."
        ),
        depth=(
            "Do not reference file names, code patterns, or specific technologies "
            "unless absolutely necessary. Focus on what the finding means for the organization."
        ),
        recommendations=(
            "Frame recommendations as business decisions with estimated cost, timeline, "
            "and risk reduction. Use language like 'invest in' not 'implement'."
        ),
        metrics=(
            "Use business metrics: estimated cost to fix, risk level (High/Medium/Low), "
            "compliance impact, timeline to remediate."
        ),
        sample=(
            "The portfolio carries significant unaddressed risk. Student data is "
            "accessible without authorization, and changes reach production with no safety checks."
        ),
    ),
    "executive": AudienceConfig(
        key="executive",
        title="Executive / VP",
        language=(
            "Use business-technical hybrid language. Technical terms are acceptable if "
            "they're commonly understood by engineering leaders. Avoid deep implementation details."
        ),
        depth=(
            "Name technologies and architectural patterns but don't go to file/function level. "
            "Quantify technical debt in terms of effort and risk."
        ),
        recommendations=(
            "Frame as strategic initiatives with team allocation and quarterly timeline. "
            "Include trade-offs."
        ),
        metrics=(
            "Use portfolio-level metrics: repo health scores, vulnerability counts, "
            "coverage percentages, effort estimates in person-weeks."
        ),
        sample=(
            "The four Supabase-backed applications bypass Row Level Security via service-role keys, "
            "creating a systemic access control gap that affects the entire Next.js portfolio."
        ),
    ),
    "technical_leadership": AudienceConfig(
        key="technical_leadership",
        title="Technical Leadership",
        language=(
            "Professional technical writing. Assume the reader understands software architecture, "
            "CI/CD, and cloud infrastructure."
        ),
        depth=(
            "Include specific technologies, version numbers, architectural patterns, and integration points. "
            "Reference specific repos when they diverge from the norm."
        ),
        recommendations=(
            "Provide prioritized remediation with effort estimates (T-shirt sizing) and dependency chains. "
            "Flag quick wins separately from structural changes."
        ),
        metrics=(
            "Include code-level metrics: complexity scores, dependency counts, test ratios, "
            "OWASP finding counts by severity."
        ),
        sample=(
            "All four Next.js applications use createClient with SUPABASE_SERVICE_KEY instead of the "
            "SSR-aware client, bypassing RLS. This pattern appears 18 times in Fleetly alone."
        ),
    ),
    "developer": AudienceConfig(
        key="developer",
        title="Developer",
        language="Direct technical language. Use framework-specific terminology freely.",
        depth=(
            "Full detail: file paths, function names, code patterns, specific CVE numbers, "
            "exact configuration changes needed."
        ),
        recommendations=(
            "Provide specific fix instructions: which file to change, what to add, code examples "
            "where helpful. Ordered by priority."
        ),
        metrics=(
            "Raw metrics: line counts, function counts, dependency versions, specific vulnerability IDs."
        ),
        sample=(
            "Fix app/api/application/lookup/route.ts L5-37: add session check before the RPC call. "
            "The parseApplicationId scramble is reversible — parameters are in the public JS bundle."
        ),
    ),
    "compliance": AudienceConfig(
        key="compliance",
        title="Compliance / Audit",
        language=(
            "Formal, evidence-based language suitable for regulatory documentation. "
            "Use passive voice where appropriate for findings."
        ),
        depth=(
            "Enough technical detail to substantiate each finding, but always framed in terms "
            "of the applicable control framework."
        ),
        recommendations=(
            "Map each recommendation to a specific control requirement. Include evidence "
            "expectations for demonstrating remediation."
        ),
        metrics=(
            "Control coverage percentages, finding counts by severity mapped to control categories, "
            "evidence inventory completeness."
        ),
        sample=(
            "Finding: No automated testing controls exist across the application portfolio. This "
            "represents a gap in change management controls per FERPA 34 CFR 99.31 safeguard requirements."
        ),
    ),
    "non_technical": AudienceConfig(
        key="non_technical",
        title="Non-Technical Stakeholder",
        language=(
            "Accessible, jargon-free language. Explain technical concepts with everyday analogies. "
            "Use 'the system' not 'the application server'."
        ),
        depth=(
            "Focus on outcomes and user-facing impact. Avoid internal architecture details entirely."
        ),
        recommendations=(
            "Frame in terms of what the team needs (time, people, budget) and what stakeholders "
            "will see when it's done."
        ),
        metrics="Simple risk ratings (Red/Yellow/Green), estimated timeline, user impact scope.",
        sample=(
            "Think of it like a building with no locks on some doors. Anyone who finds the right "
            "hallway can walk into rooms with student records. The fix takes days, not months."
        ),
    ),
}


def get_audience(key: str) -> AudienceConfig:
    """Return the audience config for a key, defaulting to technical_leadership."""
    return AUDIENCES.get(key, AUDIENCES["technical_leadership"])


def audience_prompt_block(key: str) -> str:
    """Return the prompt instructions block for an audience."""
    aud = get_audience(key)
    return (
        f"**Target Audience: {aud.title}**\n\n"
        f"Language: {aud.language}\n\n"
        f"Detail level: {aud.depth}\n\n"
        f"Recommendations style: {aud.recommendations}\n\n"
        f"Metrics style: {aud.metrics}"
    )

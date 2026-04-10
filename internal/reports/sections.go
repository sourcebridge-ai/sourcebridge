// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 SourceBridge Contributors

package reports

// ArchitectureBaselineSections defines all sections for the flagship report type.
var ArchitectureBaselineSections = []SectionDefinition{
	// Executive
	{Key: "executive_summary", Title: "Executive Summary", Category: "Executive", Description: "AI synthesis of all selected sections", DependsOn: []string{"*"}, DataSources: []string{"all_sections"}, MinWordCount: 300},
	{Key: "overall_assessment", Title: "Overall Assessment", Category: "Executive", Description: "High-level portfolio assessment with understanding scores", DependsOn: []string{"*"}, DataSources: []string{"understanding_scores", "all_sections"}, MinWordCount: 200},
	// Portfolio
	{Key: "applications_inventory", Title: "Applications Inventory", Category: "Portfolio", Description: "Repository metadata, file counts, languages", DataSources: []string{"repo_metadata"}, MinWordCount: 150},
	{Key: "application_access", Title: "Application Access", Category: "Portfolio", Description: "Platform access patterns inferred from code", DataSources: []string{"auth_detection"}, MinWordCount: 100},
	// Security
	{Key: "owasp_findings", Title: "OWASP Security Findings", Category: "Security", Description: "Automated OWASP scan results", DataSources: []string{"owasp_scan"}, MinWordCount: 200},
	{Key: "vulnerability_management", Title: "Software Vulnerability Management", Category: "Security", Description: "Dependency audit results", DataSources: []string{"dependency_audit"}, MinWordCount: 150},
	{Key: "supply_chain", Title: "Supply Chain Vulnerabilities", Category: "Security", Description: "Package analysis and supply chain risk", DataSources: []string{"dependency_audit", "license_scan"}, MinWordCount: 100},
	{Key: "user_info_protection", Title: "Protecting User Information", Category: "Security", Description: "TLS analysis, data handling patterns", DataSources: []string{"auth_detection", "secret_scanner"}, MinWordCount: 100},
	{Key: "compliance", Title: "Compliance Considerations", Category: "Security", Description: "Compliance framework applicability", DataSources: []string{"auth_detection", "repo_metadata"}, MinWordCount: 200},
	// Access
	{Key: "authentication", Title: "Authentication (AAA)", Category: "Access", Description: "Auth pattern detection", DataSources: []string{"auth_detection"}, MinWordCount: 150},
	{Key: "authorization", Title: "Authorization", Category: "Access", Description: "Role/permission pattern analysis", DataSources: []string{"auth_detection"}, MinWordCount: 100},
	{Key: "audit_trail", Title: "Audit Trail", Category: "Access", Description: "Logging pattern analysis", DataSources: []string{"auth_detection", "cicd_detection"}, MinWordCount: 100},
	// Operations
	{Key: "system_availability", Title: "System Availability", Category: "Operations", Description: "Infrastructure pattern analysis", DataSources: []string{"cicd_detection", "repo_metadata"}, MinWordCount: 100},
	{Key: "incident_response", Title: "Incident Response", Category: "Operations", Description: "Runbook/doc detection", DataSources: []string{"repo_metadata"}, MinWordCount: 80},
	{Key: "deployment_architecture", Title: "Deployment Architecture", Category: "Operations", Description: "Dockerfile, CI/CD, cloud config analysis", DataSources: []string{"cicd_detection"}, MinWordCount: 200},
	{Key: "application_secrets", Title: "Application Secrets", Category: "Operations", Description: "Secret pattern detection", DataSources: []string{"secret_scanner"}, MinWordCount: 100},
	// Delivery
	{Key: "requirements_pm", Title: "Requirements and Project Management", Category: "Delivery", Description: "Tool detection", DataSources: []string{"repo_metadata"}, MinWordCount: 80},
	{Key: "testing", Title: "Testing", Category: "Delivery", Description: "Test file detection, coverage inference", DataSources: []string{"test_detection"}, MinWordCount: 150},
	{Key: "build_deployment", Title: "Build and Deployment", Category: "Delivery", Description: "CI/CD pipeline analysis", DataSources: []string{"cicd_detection"}, MinWordCount: 150},
	{Key: "monitoring_config", Title: "Monitoring and Configuration", Category: "Delivery", Description: "Observability tool detection", DataSources: []string{"cicd_detection", "repo_metadata"}, MinWordCount: 80},
	{Key: "bugfixes_patches", Title: "Bugfixes and Patches", Category: "Delivery", Description: "Git history analysis", DataSources: []string{"git_analysis"}, MinWordCount: 100},
	// Governance
	{Key: "ownership", Title: "Application Ownership and Responsibility", Category: "Governance", Description: "CODEOWNERS, contributor analysis", DataSources: []string{"git_analysis"}, MinWordCount: 100},
	{Key: "admin_control", Title: "Access and Administrative Control", Category: "Governance", Description: "Platform access analysis", DataSources: []string{"auth_detection", "repo_metadata"}, MinWordCount: 100},
	// Data
	{Key: "data_inventory", Title: "Data Inventory and Handling", Category: "Data", Description: "Schema/model analysis", DataSources: []string{"repo_metadata", "cliff_notes"}, MinWordCount: 100},
	{Key: "third_party", Title: "Third-Party Services and Dependencies", Category: "Data", Description: "Package analysis", DataSources: []string{"dependency_audit"}, MinWordCount: 100},
	{Key: "secrets_management", Title: "Secrets and Credential Management", Category: "Data", Description: "Secret pattern analysis", DataSources: []string{"secret_scanner"}, MinWordCount: 100},
	{Key: "backup_recovery", Title: "Backup, Recovery, and Resilience", Category: "Data", Description: "Infrastructure config analysis", DataSources: []string{"cicd_detection", "repo_metadata"}, MinWordCount: 80},
	// Observability
	{Key: "logging_monitoring", Title: "Logging, Monitoring, and Observability", Category: "Observability", Description: "Log library detection", DataSources: []string{"repo_metadata", "cicd_detection"}, MinWordCount: 100},
	// Integration
	{Key: "integrations", Title: "Integration and System Interactions", Category: "Integration", Description: "API call detection, shared DB detection", DataSources: []string{"repo_metadata", "cliff_notes"}, MinWordCount: 100},
	// Engineering
	{Key: "source_control", Title: "Source Control and Development Practices", Category: "Engineering", Description: "Git workflow analysis", DataSources: []string{"git_analysis"}, MinWordCount: 150},
	{Key: "documentation", Title: "Documentation and Knowledge Management", Category: "Engineering", Description: "Doc file detection, README analysis", DataSources: []string{"repo_metadata"}, MinWordCount: 100},
	{Key: "app_lifecycle", Title: "Application Lifecycle and Maintenance", Category: "Engineering", Description: "Dependency age, update frequency", DataSources: []string{"dependency_audit", "git_analysis"}, MinWordCount: 100},
	// Users
	{Key: "system_usage", Title: "System Usage and User Base", Category: "Users", Description: "Inferred from auth + UI patterns", DataSources: []string{"auth_detection", "repo_metadata"}, MinWordCount: 80},
	{Key: "training", Title: "Training and User Enablement", Category: "Users", Description: "Doc/guide detection", DataSources: []string{"repo_metadata"}, MinWordCount: 80},
	// Review
	{Key: "arch_review_findings", Title: "Architecture Review Key Findings", Category: "Review", DependsOn: []string{"owasp_findings", "testing", "deployment_architecture", "source_control"}, Description: "AI deep analysis synthesis", DataSources: []string{"all_sections"}, MinWordCount: 400},
	// Appendix
	{Key: "appendix_team", Title: "Team Members", Category: "Appendix", Description: "Git contributor extraction", DataSources: []string{"git_analysis"}},
	{Key: "appendix_owasp", Title: "OWASP Security Scan Results", Category: "Appendix", Description: "Per-repo detailed findings", DataSources: []string{"owasp_scan"}},
	{Key: "appendix_deps", Title: "Dependency Audit Results", Category: "Appendix", Description: "Per-repo audit output", DataSources: []string{"dependency_audit"}},
}

// SWOTSections defines sections for the SWOT report type.
var SWOTSections = []SectionDefinition{
	{Key: "strengths", Title: "Strengths", Category: "Analysis", Description: "What the codebase does well", DataSources: []string{"cliff_notes", "understanding_scores", "test_detection"}, MinWordCount: 200},
	{Key: "weaknesses", Title: "Weaknesses", Category: "Analysis", Description: "Technical debt, gaps, risks", DataSources: []string{"dependency_audit", "owasp_scan", "test_detection"}, MinWordCount: 200},
	{Key: "opportunities", Title: "Opportunities", Category: "Analysis", Description: "Improvements and modernization paths", DataSources: []string{"dependency_audit", "repo_metadata", "cliff_notes"}, MinWordCount: 200},
	{Key: "threats", Title: "Threats", Category: "Analysis", Description: "External and internal risks", DataSources: []string{"owasp_scan", "dependency_audit", "git_analysis"}, MinWordCount: 200},
	{Key: "recommendations", Title: "Recommendations", Category: "Synthesis", DependsOn: []string{"strengths", "weaknesses", "opportunities", "threats"}, Description: "Prioritized action items", DataSources: []string{"all_sections"}, MinWordCount: 300},
}

// EnvironmentEvalSections defines sections for the Environment Evaluation report type.
var EnvironmentEvalSections = []SectionDefinition{
	{Key: "tech_stack", Title: "Technology Stack Summary", Category: "Overview", Description: "Languages, frameworks, databases, cloud services", DataSources: []string{"repo_metadata", "dependency_audit"}, MinWordCount: 200},
	{Key: "infrastructure", Title: "Infrastructure Topology", Category: "Overview", Description: "Deployment targets, hosting, networking patterns", DataSources: []string{"cicd_detection"}, MinWordCount: 150},
	{Key: "dev_toolchain", Title: "Development Toolchain", Category: "Overview", Description: "Build tools, linters, formatters, CI/CD", DataSources: []string{"cicd_detection", "repo_metadata"}, MinWordCount: 100},
	{Key: "code_quality", Title: "Code Quality Metrics", Category: "Quality", Description: "Complexity, test ratios, doc coverage", DataSources: []string{"understanding_scores", "test_detection"}, MinWordCount: 150},
	{Key: "security_posture", Title: "Security Posture", Category: "Security", Description: "Aggregated vulnerability findings", DataSources: []string{"owasp_scan", "dependency_audit"}, MinWordCount: 150},
	{Key: "operational_maturity", Title: "Operational Maturity", Category: "Operations", Description: "Monitoring, logging, alerting, runbooks", DataSources: []string{"cicd_detection", "repo_metadata"}, MinWordCount: 100},
	{Key: "integration_map", Title: "Integration Map", Category: "Integration", Description: "How systems connect", DataSources: []string{"repo_metadata", "cliff_notes"}, MinWordCount: 100},
	{Key: "compliance_readiness", Title: "Compliance Readiness", Category: "Compliance", Description: "Gap analysis against common frameworks", DataSources: []string{"auth_detection", "repo_metadata"}, MinWordCount: 150},
	{Key: "cost_resources", Title: "Cost and Resource Implications", Category: "Resources", Description: "Cloud resource patterns", DataSources: []string{"cicd_detection", "repo_metadata"}, MinWordCount: 100},
	{Key: "modernization", Title: "Modernization Readiness", Category: "Modernization", Description: "Framework versions, migration complexity", DataSources: []string{"dependency_audit", "repo_metadata"}, MinWordCount: 150},
}

// PortfolioHealthSections defines sections for the Portfolio Health Dashboard report.
var PortfolioHealthSections = []SectionDefinition{
	{Key: "portfolio_overview", Title: "Portfolio Overview", Category: "Overview", Description: "Repo count, total LOC, languages, activity", DataSources: []string{"repo_metadata"}, MinWordCount: 150},
	{Key: "understanding_scores", Title: "Understanding Scores", Category: "Scores", Description: "Score breakdown per repo", DataSources: []string{"understanding_scores"}, MinWordCount: 100},
	{Key: "security_summary", Title: "Security Summary", Category: "Security", Description: "Aggregated findings", DataSources: []string{"owasp_scan", "dependency_audit"}, MinWordCount: 100},
	{Key: "freshness_report", Title: "Freshness Report", Category: "Freshness", Description: "Stale artifacts, outdated deps", DataSources: []string{"dependency_audit", "repo_metadata"}, MinWordCount: 100},
	{Key: "activity_summary", Title: "Activity Summary", Category: "Activity", Description: "Recent commits, active contributors", DataSources: []string{"git_analysis"}, MinWordCount: 100},
	{Key: "cross_repo_patterns", Title: "Cross-Repo Patterns", Category: "Patterns", Description: "Shared libraries, inconsistencies", DataSources: []string{"repo_metadata", "dependency_audit"}, MinWordCount: 150},
}

// DueDiligenceSections defines sections for the Technical Due Diligence report.
var DueDiligenceSections = []SectionDefinition{
	{Key: "dd_executive_summary", Title: "Executive Summary", Category: "Executive", DependsOn: []string{"*"}, Description: "Investment-grade overview", DataSources: []string{"all_sections"}, MinWordCount: 300},
	{Key: "tech_risk", Title: "Technology Risk Assessment", Category: "Risk", Description: "Technical debt quantification", DataSources: []string{"dependency_audit", "owasp_scan", "test_detection"}, MinWordCount: 200},
	{Key: "scalability", Title: "Scalability Analysis", Category: "Architecture", Description: "Architecture scaling patterns", DataSources: []string{"cliff_notes", "repo_metadata"}, MinWordCount: 150},
	{Key: "team_knowledge_risk", Title: "Team and Knowledge Risk", Category: "People", Description: "Bus factor, documentation gaps", DataSources: []string{"git_analysis", "repo_metadata"}, MinWordCount: 150},
	{Key: "ip_licensing", Title: "IP and Licensing", Category: "Legal", Description: "License analysis across dependencies", DataSources: []string{"license_scan"}, MinWordCount: 100},
	{Key: "dd_security", Title: "Security and Compliance", Category: "Security", Description: "Vulnerability and compliance summary", DataSources: []string{"owasp_scan", "dependency_audit"}, MinWordCount: 200},
	{Key: "operational_continuity", Title: "Operational Continuity", Category: "Operations", Description: "Deployment, monitoring, incident readiness", DataSources: []string{"cicd_detection", "repo_metadata"}, MinWordCount: 150},
	{Key: "remediation_effort", Title: "Estimated Remediation Effort", Category: "Effort", DependsOn: []string{"tech_risk", "dd_security"}, Description: "Prioritized backlog with sizing", DataSources: []string{"all_sections"}, MinWordCount: 200},
}

// ComplianceGapSections defines sections for the Compliance Gap Analysis report.
var ComplianceGapSections = []SectionDefinition{
	{Key: "framework_overview", Title: "Framework Overview", Category: "Framework", Description: "Selected compliance framework summary", DataSources: []string{"repo_metadata"}, MinWordCount: 150},
	{Key: "control_mapping", Title: "Control Mapping", Category: "Controls", Description: "Map code evidence to framework controls", DataSources: []string{"auth_detection", "test_detection", "cicd_detection"}, MinWordCount: 300},
	{Key: "gap_identification", Title: "Gap Identification", Category: "Gaps", Description: "Controls with insufficient evidence", DataSources: []string{"auth_detection", "test_detection", "cicd_detection"}, MinWordCount: 200},
	{Key: "risk_rating", Title: "Risk Rating", Category: "Risk", Description: "Per-control risk assessment", DataSources: []string{"owasp_scan", "dependency_audit"}, MinWordCount: 150},
	{Key: "remediation_roadmap", Title: "Remediation Roadmap", Category: "Remediation", DependsOn: []string{"gap_identification", "risk_rating"}, Description: "Prioritized steps to close gaps", DataSources: []string{"all_sections"}, MinWordCount: 200},
	{Key: "evidence_inventory", Title: "Evidence Inventory", Category: "Evidence", Description: "What evidence exists in the codebase", DataSources: []string{"auth_detection", "test_detection", "cicd_detection", "repo_metadata"}, MinWordCount: 100},
}

// AllReportTypes returns definitions for all report types.
func AllReportTypes() []ReportTypeDefinition {
	return []ReportTypeDefinition{
		{Type: TypeArchitectureBaseline, Title: "Software Architecture Baseline", Description: "Comprehensive technical review of your software portfolio covering architecture, security, operations, and organizational readiness.", Sections: ArchitectureBaselineSections},
		{Type: TypeSWOT, Title: "SWOT Analysis", Description: "Strategic assessment of strengths, weaknesses, opportunities, and threats across your codebase.", Sections: SWOTSections},
		{Type: TypeEnvironmentEval, Title: "Environment Evaluation", Description: "Assessment of the overall technology environment including stack, infrastructure, quality, and compliance readiness.", Sections: EnvironmentEvalSections},
		{Type: TypePortfolioHealth, Title: "Portfolio Health Dashboard", Description: "Periodic summary of all indexed repositories with scores, security, freshness, and activity.", Sections: PortfolioHealthSections},
		{Type: TypeDueDiligence, Title: "Technical Due Diligence", Description: "Investment-grade assessment for M&A, vendor evaluation, or technology risk analysis.", Sections: DueDiligenceSections},
		{Type: TypeComplianceGap, Title: "Compliance Gap Analysis", Description: "Assessment against FERPA, HIPAA, SOC 2, PCI-DSS, or NIST with control mapping and remediation roadmap.", Sections: ComplianceGapSections},
	}
}

// SectionsForType returns the section definitions for a report type.
func SectionsForType(rt ReportType) []SectionDefinition {
	for _, def := range AllReportTypes() {
		if def.Type == rt {
			return def.Sections
		}
	}
	return nil
}

// DefaultSectionsForAudience returns the recommended section keys for a
// given report type + audience combination. This powers the "smart
// defaults" in the wizard.
func DefaultSectionsForAudience(rt ReportType, aud Audience) []string {
	sections := SectionsForType(rt)
	if sections == nil {
		return nil
	}

	var keys []string
	for _, sec := range sections {
		include := true
		switch aud {
		case AudienceCSuite:
			// C-Suite: executive, security overview, compliance, assessment — skip deep engineering
			switch sec.Category {
			case "Engineering", "Appendix":
				include = false
			}
		case AudienceNonTechnical:
			// Non-technical: executive, overview, recommendations — skip detailed technical
			switch sec.Category {
			case "Engineering", "Appendix", "Data", "Observability", "Integration":
				include = false
			}
		case AudienceCompliance:
			// Compliance: security, access, governance, data, appendix — skip engineering details
			switch sec.Category {
			case "Engineering", "Users":
				include = false
			}
		default:
			// Technical audiences get everything
		}
		if include {
			keys = append(keys, sec.Key)
		}
	}
	return keys
}

export type SafetyLevel = "Stable" | "Elevated" | "High Risk" | "Critical" | "Data Unreliable";

export interface IncidentRow {
  id: string;
  symbols: string[];
  venue: string;
  countryIso2: string;
  countryCode: string;
  countryName: string;
  region: string;
  sector: string;
  industry: string;
  severityBand: string;
  safetyLevel: SafetyLevel;
  compositeRisk: number;
  escalationProbability: number;
  confidence: number;
  trustPenalty: number;
  priorityScore: number;
  recommendedAction: string;
  topDrivers: { feature: string; contribution: number; deltaVsBaseline: number }[];
  owner?: string;
  ageSeconds: number;
  status: string;
}

export type CountryAgg = {
  countryIso2: string;
  countryCode: string;
  countryName: string;
  incidentCount: number;
  avgCompositeRisk: number;
  maxSafetyLevel: SafetyLevel;
  topIndustry?: string;
  trustState: "healthy" | "degraded" | "unreliable";
};

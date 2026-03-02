import type { GetHomeSummaryInput, HomeSummary } from "@/features/home/domain/HomeSummary";

export interface HomeRepository {
  getSummary(input: GetHomeSummaryInput): Promise<HomeSummary>;
}

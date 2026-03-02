import type { GetHomeSummaryInput } from "@/features/home/domain/HomeSummary";
import type { HomeRepository } from "@/features/home/repository/HomeRepository";

export class BrowseHomeUseCase {
  constructor(private readonly repo: HomeRepository) {}

  getSummary(input: GetHomeSummaryInput) {
    if (!input.start || !input.end) {
      throw new Error("start/end is required");
    }
    return this.repo.getSummary(input);
  }
}

import { api } from "@/features/shared/infrastructure/http-api";
import { getAccessToken } from "@/features/auth/infrastructure/token-auth";
import type { GetHomeSummaryInput, HomeSummary } from "@/features/home/domain/HomeSummary";
import type { HomeRepository } from "@/features/home/repository/HomeRepository";

export class HttpHomeRepository implements HomeRepository {
  private getToken(): string {
    const token = getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  getSummary(input: GetHomeSummaryInput): Promise<HomeSummary> {
    const token = this.getToken();
    const params = new URLSearchParams({
      start: input.start,
      end: input.end,
    });

    if (input.event_limit) params.set("event_limit", String(input.event_limit));
    if (input.todo_limit) params.set("todo_limit", String(input.todo_limit));
    if (input.recent_limit) params.set("recent_limit", String(input.recent_limit));

    return api<HomeSummary>(`/home/summary?${params.toString()}`, { token });
  }
}

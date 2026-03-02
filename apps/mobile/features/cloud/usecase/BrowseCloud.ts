import type { CloudRepository } from "../repository/CloudRepository";

export class BrowseCloudUseCase {
  constructor(private readonly repo: CloudRepository) {}

  listItems(folderId?: string | null) {
    return this.repo.listItems(folderId);
  }
}

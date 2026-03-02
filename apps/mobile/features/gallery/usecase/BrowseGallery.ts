import type { GalleryRepository } from "../repository/GalleryRepository";

export class BrowseGalleryUseCase {
  constructor(private readonly repo: GalleryRepository) {}

  listMedia() {
    return this.repo.listMedia();
  }
}

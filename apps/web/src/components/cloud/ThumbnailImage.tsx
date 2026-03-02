"use client";

import Image from "next/image";
import type { ThumbSize } from "@/features/gallery/domain/MediaFile";
import { useThumbnailSource } from "@/features/gallery/ui/hooks/useThumbnailSource";

type ThumbnailImageProps =
  | {
      fileId: string;
      size: ThumbSize;
      alt: string;
      className?: string;
      sizes?: string;
      fallback?: React.ReactNode;
      fill: true;
      width?: never;
      height?: never;
    }
  | {
      fileId: string;
      size: ThumbSize;
      alt: string;
      className?: string;
      sizes?: string;
      fallback?: React.ReactNode;
      fill?: false;
      width: number;
      height: number;
    };

export function ThumbnailImage(props: ThumbnailImageProps) {
  const { fileId, size, alt, className, sizes, fallback } = props;
  const src = useThumbnailSource(fileId, size);

  if (!src) return <>{fallback ?? null}</>;

  if (props.fill) {
    return (
      <Image
        src={src}
        alt={alt}
        fill
        unoptimized
        sizes={sizes}
        className={className}
      />
    );
  }

  return (
    <Image
      src={src}
      alt={alt}
      width={props.width}
      height={props.height}
      unoptimized
      sizes={sizes}
      className={className}
    />
  );
}

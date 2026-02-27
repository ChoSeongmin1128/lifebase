"use client";

import Image from "next/image";
import { useState, useEffect } from "react";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:38117";

type ThumbSize = "small" | "medium";

type ThumbnailImageProps =
  | {
      fileId: string;
      size: ThumbSize;
      token: string | null;
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
      token: string | null;
      alt: string;
      className?: string;
      sizes?: string;
      fallback?: React.ReactNode;
      fill?: false;
      width: number;
      height: number;
    };

export function ThumbnailImage(props: ThumbnailImageProps) {
  const { fileId, size, token, alt, className, sizes, fallback } = props;
  const [src, setSrc] = useState<string | null>(null);

  useEffect(() => {
    if (!token) {
      setSrc(null);
      return;
    }

    const controller = new AbortController();
    let objectURL: string | null = null;
    let active = true;

    const load = async () => {
      try {
        const res = await fetch(`${API_URL}/api/v1/gallery/thumbnails/${fileId}/${size}`, {
          headers: { Authorization: `Bearer ${token}` },
          signal: controller.signal,
        });
        if (!res.ok) throw new Error(`thumbnail ${res.status}`);

        const blob = await res.blob();
        objectURL = URL.createObjectURL(blob);
        if (active) setSrc(objectURL);
      } catch {
        if (active) setSrc(null);
      }
    };

    load();

    return () => {
      active = false;
      controller.abort();
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [fileId, size, token]);

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

"use client";

import { useState, useCallback } from "react";

const STORAGE_KEY = "lifebase-sidebar";

function getStoredState(): boolean {
  if (typeof window === "undefined") return true;
  const stored = localStorage.getItem(STORAGE_KEY);
  return stored !== "collapsed";
}

export function useSidebar() {
  const [expanded, setExpanded] = useState(getStoredState);

  const toggle = useCallback(() => {
    setExpanded((prev) => {
      const next = !prev;
      localStorage.setItem(STORAGE_KEY, next ? "expanded" : "collapsed");
      return next;
    });
  }, []);

  return { expanded, toggle };
}

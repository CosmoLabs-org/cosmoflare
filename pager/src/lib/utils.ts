import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

/** shadcn/ui's cn: conditional classes, Tailwind-aware merge. */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

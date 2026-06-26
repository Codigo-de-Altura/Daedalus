/** Minimal className joiner — truthy values only, space-separated. */
export function cn(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(" ");
}

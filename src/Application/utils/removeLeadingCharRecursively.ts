export const removeLeadingCharRecursively = (s: string, char: string): string =>
  s[0] === char ? removeLeadingCharRecursively(s.substr(1), char) : s

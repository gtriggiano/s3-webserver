export const removeTrailingCharRecursively = (
  s: string,
  char: string,
): string =>
  s[s.length - 1] === char
    ? removeTrailingCharRecursively(s.substr(0, s.length - 1), char)
    : s

import { removeLeadingCharRecursively } from './removeLeadingCharRecursively'

export const getFolderHTML = (
  folderPath: string,
  childrenPaths: string[],
): string => {
  const items = childrenPaths.reduce<string>((items, path) => {
    const displayName = removeLeadingCharRecursively(
      path.replace(folderPath, ''),
      '/',
    )
    const item = `<li><a href="${path}">${displayName}</a></li>`
    return `${items}${item}`
  }, '')

  return `<html>
<head>
  <title>${folderPath}</title>
</head>
<body>
  <h1>${folderPath}</h1>
  <ul>${items}</ul>
</body>
</html>`
}

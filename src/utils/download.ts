// Hands a downloaded Blob to the browser as a file save. Without a filename the
// browser picks its own.
export function saveBlob(data: Blob, filename?: string) {
  const url = URL.createObjectURL(data)
  const link = document.createElement('a')
  link.href = url
  link.download = filename ?? ''
  link.click()
  URL.revokeObjectURL(url)
}

// The filename the server suggested in a Content-Disposition header, if any.
export function dispositionFilename(response: Response): string | undefined {
  return /filename="([^"]+)"/.exec(response.headers.get('Content-Disposition') ?? '')?.[1]
}

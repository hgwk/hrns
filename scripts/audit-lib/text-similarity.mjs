const DEFAULT_STOP_WORDS = new Set([
  'the',
  'and',
  'for',
  'with',
  'that',
  'this',
  'from',
  '으로',
  '에서',
  '하는',
  '그리고',
  '또는',
])

export function tokenSet(text, extraStopWords = []) {
  const stop = new Set([...DEFAULT_STOP_WORDS, ...extraStopWords])
  return new Set(
    (text.toLowerCase().match(/[a-z0-9가-힣_:-]{3,}/g) ?? []).filter((token) => !stop.has(token)),
  )
}

export function jaccard(left, right) {
  let intersection = 0
  for (const token of left) if (right.has(token)) intersection += 1
  return intersection / Math.max(1, left.size + right.size - intersection)
}

export function firstHeading(text) {
  const match = text.match(/^#\s+(.+)$/m)
  return match ? match[1].trim() : ''
}

export function headings(text) {
  return [...text.matchAll(/^#{1,4}\s+(.+)$/gm)].map((match) => match[1].trim())
}

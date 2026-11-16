// Match backend/internal/wiki/indexer/chunks.go, using raw ATX titles.
export function headingAnchors(content: string): Map<number, string> {
  const ids = new Map<number, string>();
  const occurrences = new Map<string, number>();
  let fence = '';
  let fenceLength = 0;
  content
    .replace(/\r\n/g, '\n')
    .split('\n')
    .forEach((line, index) => {
      const trimmed = line.replace(/^ {0,3}/, '');
      const marker = trimmed.match(/^(`+|~+)/)?.[0];
      if (marker) {
        if (!fence && marker.length >= 3) {
          fence = marker[0];
          fenceLength = marker.length;
        } else if (
          fence === marker[0] &&
          marker.length >= fenceLength &&
          !trimmed.slice(marker.length).trim()
        )
          fence = '';
      }
      if (fence) return;
      const heading = line.match(/^ {0,3}(#{1,6})(?:[ \t]+(.*)|$)/);
      if (!heading) return;
      let title = (heading[2] || '').trim();
      title = title.replace(/(?:^|[ \t]+)#+$/, '').trim();
      const base = title
        .toLowerCase()
        .replace(/ /g, '-')
        .replace(/[^\p{L}\p{N}\p{M}_-]/gu, '');
      let anchor = base;
      while (occurrences.has(anchor)) {
        const count = (occurrences.get(base) || 0) + 1;
        occurrences.set(base, count);
        anchor = `${base}-${count}`;
      }
      occurrences.set(anchor, 0);
      ids.set(index + 1, anchor);
    });
  return ids;
}

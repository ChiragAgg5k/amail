const day = 86400000;
const windows = [3, 14, 60, 365, 3650];

function run(argv) {
  const request = JSON.parse(argv[0]);
  const messages = findMailbox(request).messages;
  const fixed = request.unread ? [{ readStatus: false }] : [];

  // Asking Mail for one property of many messages is a single Apple event, while
  // walking messages by position costs several events each. Widening a date window
  // until it holds enough messages keeps large mailboxes fast.
  for (const days of windows) {
    const since = new Date(Date.now() - days * day);
    const rows = summaries(filtered(messages, fixed.concat([{ dateReceived: { _greaterThan: since } }])));
    if (rows.length >= request.limit) return JSON.stringify(newest(rows, request.limit));
  }
  return JSON.stringify(newest(summaries(filtered(messages, fixed)), request.limit));
}

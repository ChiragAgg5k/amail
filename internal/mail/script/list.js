function main(request) {
  const day = 86400000;
  const windows = [3, 14, 60, 365, 3650];
  const mailboxName = request.mailbox || 'Inbox';
  const messages = findMailbox({ account: request.account, mailbox: mailboxName }).messages;
  const fixed = request.unread ? [{ readStatus: false }] : [];

  // Widening a date window until it holds enough messages keeps large mailboxes fast,
  // since every bulk property read costs time in proportion to the rows it returns.
  for (const days of windows) {
    const since = new Date(Date.now() - days * day);
    const rows = summaries(filtered(messages, fixed.concat([{ dateReceived: { _greaterThan: since } }])), mailboxName);
    if (request.limit > 0 && rows.length >= request.limit) return newest(rows, request.limit);
  }
  return newest(summaries(filtered(messages, fixed), mailboxName), request.limit);
}

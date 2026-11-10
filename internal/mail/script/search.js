function main(request) {
  const located = locateMailbox({ account: request.account, mailbox: request.mailbox || 'Inbox' });
  const mailboxName = located.name;
  const messages = located.mailbox.messages;
  const conditions = [];
  if (request.from) conditions.push({ sender: { _contains: request.from } });
  if (request.subject) conditions.push({ subject: { _contains: request.subject } });
  if (request.unread) conditions.push({ readStatus: false });
  if (request.since) conditions.push({ dateReceived: { _greaterThan: new Date(request.since) } });

  if (!request.body || !request.text) {
    if (request.text) conditions.push({ _or: [{ subject: { _contains: request.text } }, { sender: { _contains: request.text } }] });
    if (request.since) return newest(summaries(filtered(messages, conditions), mailboxName), request.limit);
    return recent(messages, conditions, mailboxName, request.limit);
  }

  // Mail cannot filter on body text, so each body costs its own content() read
  // (about 300 ms). The mandatory date bound keeps that set small, and rows whose
  // subject or sender already match never have their body read.
  if (!request.since) throw new Error('searching message bodies needs a since date');
  const text = request.text.toLowerCase();
  const matches = [];
  for (const row of newest(summaries(filtered(messages, conditions), mailboxName), 0)) {
    if (request.limit > 0 && matches.length >= request.limit) break;
    if ((row.subject + '\n' + row.sender).toLowerCase().indexOf(text) >= 0) {
      matches.push(row);
      continue;
    }
    const content = optional(() => messages.byId(row.id).content()) || '';
    if (String(content).toLowerCase().indexOf(text) >= 0) matches.push(row);
  }
  return matches;
}

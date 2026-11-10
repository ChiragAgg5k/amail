function main(request) {
  const located = locateMailbox({ account: request.account, mailbox: request.mailbox || 'Inbox' });
  const conditions = request.unread ? [{ readStatus: false }] : [];
  return recent(located.mailbox.messages, conditions, located.name, request.limit);
}

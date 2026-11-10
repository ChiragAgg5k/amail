function main(request) {
  const located = locateMailbox({ account: request.account, mailbox: request.mailbox || 'Inbox' });
  return matching(located.mailbox.messages, { unread: request.unread }, located.name, request.limit);
}

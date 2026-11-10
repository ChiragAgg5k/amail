function main(request) {
  return preludeEntries(request.account).map((entry) => ({
    name: entry.name,
    unread: optional(() => entry.mailbox.unreadCount()) || 0,
    messages: optional(() => entry.mailbox.messages.length) || 0,
  }));
}

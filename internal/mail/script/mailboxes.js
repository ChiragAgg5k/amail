function run(argv) {
  const request = JSON.parse(argv[0]);
  const mailboxes = findAccount(request.account).mailboxes;
  const names = mailboxes.name();
  const unread = mailboxes.unreadCount();
  const rows = names.map((name, position) => ({
    name: name,
    unread: unread[position],
    messages: mailboxes.byName(name).messages.length,
  }));
  return JSON.stringify(rows);
}
